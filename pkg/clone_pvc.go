/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"context"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/templates"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

var (
	cloneExample = templates.Examples(`
		# Clone PVC
		stash clone pvc source-pvc -n demo --to-namespace=demo1 --secret=<secret> --bucket=<bucket> --prefix=<prefix> --provider=<provider>`)
)

func NewCmdClonePVC() *cobra.Command {
	var repoOpt = repositoryOption{}
	var cmd = &cobra.Command{
		Use:               "pvc",
		Short:             `Clone PVC`,
		Long:              `Use Backup and Restore process for cloning PVC`,
		Example:           cloneExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("PVC name is not provided ")
			}

			pvcName := args[0]

			pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(srcNamespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// to clone a PVC from source namespace to destination namespace, Steps are following:
			// 1. create Repository to the source namespace.
			// 2. create BackupConfiguration to take backup of the source PVC.
			// 3. clone Repository to the destination namespace
			// 4. then restore the pvc to the destination namespace.

			// set unique name for a Repository and create a Repository to the source namespace
			repoName := fmt.Sprintf("%s-%s-%d", repoOpt.provider, "repo", time.Now().Unix())
			log.Infof("Creating Repository: %s to the Namespace: %s", repoName, srcNamespace)
			repository := newRepository(repoOpt, repoName, srcNamespace)
			_, err = createRepository(repository, repository.ObjectMeta)
			if err != nil {
				return err
			}
			log.Infof("Repository has been created successfully.")

			err = backupPVC(pvcName, repoName)
			if err != nil {
				return err
			}
			log.Infof("The PVC %s/%s data has been backed up successfully", pvc.Namespace, pvc.Name)

			// copy repository and secret to the destination namespace
			err = ensureRepository(repoName)
			if err != nil {
				return err
			}

			err = restorePVC(pvc, repoName)
			if err != nil {
				return err
			}
			// delete all repository
			err = cleanupRepository(repoName)
			if err != nil {
				return err
			}
			log.Infof("PVC has been cloned successfully!!")

			return nil
		},
	}
	cmd.Flags().StringVar(&repoOpt.provider, "provider", repoOpt.provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&repoOpt.bucket, "bucket", repoOpt.bucket, "Name of the cloud bucket/container")
	cmd.Flags().StringVar(&repoOpt.endpoint, "endpoint", repoOpt.endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().Int64Var(&repoOpt.maxConnections, "max-connections", repoOpt.maxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&repoOpt.secret, "secret", repoOpt.secret, "Name of the Storage Secret")
	cmd.Flags().StringVar(&repoOpt.prefix, "prefix", repoOpt.prefix, "Prefix denotes the directory inside the backend")
	return cmd
}

// at first, create BackupConfiguration to take backup
// after successful taking backup, delete the BackupConfiguration to stop taking backup
func backupPVC(pvcName string, repoName string) error {
	// configure BackupConfiguration
	opt := backupConfigOption{
		task:       "pvc-backup",
		schedule:   "*/59 * * * *", // we have to set a large value then trigger an instant backup immediately.
		repository: repoName,
		retentionPolicy: v1alpha1.RetentionPolicy{
			Name:     "keep-last-5",
			KeepLast: 5,
			Prune:    true,
		},
		targetRef: v1beta1.TargetRef{
			Name:       pvcName,
			Kind:       apis.KindPersistentVolumeClaim,
			APIVersion: core.SchemeGroupVersion.String(),
		},
	}
	backupConfig, err := opt.newBackupConfiguration(fmt.Sprintf("%s-%s", pvcName, "backup"), srcNamespace)
	if err != nil {
		return err
	}
	log.Infof("Creating BackupConfiguration: %s to the namespace: %s", backupConfig.Name, backupConfig.Namespace)
	backupConfig, err = createBackupConfiguration(backupConfig, backupConfig.ObjectMeta)
	if err != nil {
		return err
	}
	log.Infof("BackupConfiguration has been created successfully.")

	backupSession, err := triggerBackup(backupConfig, stashClient)
	if err != nil {
		return err
	}

	err = WaitUntilBackupSessionCompleted(backupSession.Name, backupSession.Namespace)
	if err != nil {
		return err
	}
	log.Infof("BackupSession has been succeeded.")
	// delete the BackupConfiguration to stop taking backup
	return stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Delete(context.TODO(), backupConfig.Name, metav1.DeleteOptions{})
}

// create RestoreSession to create a new PVC in the destination namespace
// then restore the backed up data into the PVC
func restorePVC(pvc *core.PersistentVolumeClaim, repoName string) error {
	// configure RestoreSession
	opt := restoreSessionOption{
		repository: repoName,
		task:       "pvc-restore",
		rule: v1beta1.Rule{
			Snapshots: []string{"latest"},
		},
	}

	restoreSession, err := opt.newRestoreSession(fmt.Sprintf("%s-%s", pvc.Name, "restore"), dstNamespace)
	if err != nil {
		return err
	}
	restoreSession.Spec.Target.VolumeClaimTemplates = []ofst.PersistentVolumeClaim{
		{
			PartialObjectMeta: ofst.PartialObjectMeta{
				Name:      pvc.Name,
				Namespace: dstNamespace,
			},
			Spec: core.PersistentVolumeClaimSpec{
				StorageClassName: pvc.Spec.StorageClassName,
				Resources:        pvc.Spec.Resources,
				AccessModes:      pvc.Spec.AccessModes,
			},
		},
	}

	log.Infof("Creating RestoreSession: %s to the namespace: %s", restoreSession.Name, restoreSession.Namespace)
	restoreSession, err = createRestoreSession(restoreSession)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession has been created successfully.")
	err = WaitUntilRestoreSessionCompleted(restoreSession.Name, restoreSession.Namespace)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession has been succeeded.")
	// delete RestoreSession
	return stashClient.StashV1beta1().RestoreSessions(dstNamespace).Delete(context.TODO(), restoreSession.Name, metav1.DeleteOptions{})
}

func cleanupRepository(repoName string) error {
	err := stashClient.StashV1alpha1().Repositories(srcNamespace).Delete(context.TODO(), repoName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return stashClient.StashV1alpha1().Repositories(dstNamespace).Delete(context.TODO(), repoName, metav1.DeleteOptions{})
}
