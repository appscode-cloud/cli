/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package akp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// Domain and reasons of the google.rpc.ErrorInfo the AKP v2 API attaches to errors.
const (
	errorDomain           = "akp"
	reasonAccountInactive = "ACCOUNT_INACTIVE"
	reasonLoginProhibited = "LOGIN_PROHIBITED"
	reasonOrgSSORequired  = "ORG_SSO_REQUIRED"
)

// errorInfoInterceptor adds to an AKP v2 API error the page where the user can resolve it.
func errorInfoInterceptor(endpoint string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				err = withErrorInfo(err, endpoint)
			}
			return resp, err
		}
	}
}

// withErrorInfo wraps err, so connect.CodeOf still sees its code.
func withErrorInfo(err error, endpoint string) error {
	info := errorInfo(err)
	if info == nil {
		return err
	}
	md := info.GetMetadata()
	page := func(key string) string {
		if md[key] == "" {
			return ""
		}
		return strings.TrimSuffix(endpoint, "/") + md[key]
	}

	switch info.GetReason() {
	case reasonAccountInactive:
		if url := page("activation_path"); url != "" {
			return fmt.Errorf("%w; activate the account at %s", err, url)
		}
	case reasonLoginProhibited:
		if url := page("prohibit_login_path"); url != "" {
			return fmt.Errorf("%w; the account may not sign in, see %s", err, url)
		}
	case reasonOrgSSORequired:
		if url := page("oauth_callback_path"); url != "" {
			return fmt.Errorf("%w; organization %s requires signing in through its SSO at %s", err, md["organization"], url)
		}
	}
	return err
}

func errorInfo(err error) *errdetails.ErrorInfo {
	var ce *connect.Error
	if !errors.As(err, &ce) {
		return nil
	}
	for _, d := range ce.Details() {
		v, err := d.Value()
		if err != nil {
			continue
		}
		if info, ok := v.(*errdetails.ErrorInfo); ok && info.GetDomain() == errorDomain {
			return info
		}
	}
	return nil
}
