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
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func infoError(code connect.Code, reason string, md map[string]string) error {
	ce := connect.NewError(code, errors.New("refused"))
	d, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: reason, Domain: errorDomain, Metadata: md})
	if err != nil {
		panic(err)
	}
	ce.AddDetail(d)
	return ce
}

func TestWithErrorInfo(t *testing.T) {
	const endpoint = "https://akp.example.com/"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "account inactive",
			err:  infoError(connect.CodeFailedPrecondition, reasonAccountInactive, map[string]string{"activation_path": "/accounts/user/activate"}),
			want: "activate the account at https://akp.example.com/accounts/user/activate",
		},
		{
			name: "login prohibited",
			err:  infoError(connect.CodePermissionDenied, reasonLoginProhibited, map[string]string{"prohibit_login_path": "/accounts/user/prohibit_login"}),
			want: "see https://akp.example.com/accounts/user/prohibit_login",
		},
		{
			name: "org sso required",
			err: infoError(connect.CodePermissionDenied, reasonOrgSSORequired, map[string]string{
				"organization":        "acme",
				"oauth_callback_path": "/user/oauth2/org-3",
			}),
			want: "organization acme requires signing in through its SSO at https://akp.example.com/user/oauth2/org-3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withErrorInfo(tt.err, endpoint)
			if !strings.Contains(got.Error(), tt.want) {
				t.Errorf("got %q, want it to contain %q", got.Error(), tt.want)
			}
			if connect.CodeOf(got) != connect.CodeOf(tt.err) {
				t.Errorf("code %v, want %v", connect.CodeOf(got), connect.CodeOf(tt.err))
			}
		})
	}
}

func TestWithErrorInfoLeavesOtherErrors(t *testing.T) {
	plain := connect.NewError(connect.CodeNotFound, errors.New("no cluster"))
	if got := withErrorInfo(plain, "https://akp.example.com"); got != error(plain) {
		t.Errorf("got %v, want the error unchanged", got)
	}
	noPath := infoError(connect.CodeFailedPrecondition, reasonAccountInactive, nil)
	if got := withErrorInfo(noPath, "https://akp.example.com"); got != noPath {
		t.Errorf("got %v, want the error unchanged", got)
	}
}
