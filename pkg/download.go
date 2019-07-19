package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/cli/pkg/docker"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdDownloadRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		localDirs  = &cliLocalDirectories{}
		restoreOpt = restic.RestoreOptions{
			SourceHost:  restic.DefaultHost,
			Destination: docker.DestinationDir,
		}
	)

	var cmd = &cobra.Command{
		Use:               "download",
		Short:             `Download snapshots`,
		Long:              `Download contents of snapshots from Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name not found")
			}
			repositoryName := args[0]

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err := clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			kc, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}
			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			// get source repository
			repository, err := client.StashV1alpha1().Repositories(namespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// unlock local backend
			if repository.Spec.Backend.Local != nil {
				return fmt.Errorf("can't restore from repository with local backend")
			}
			// get repository secret
			secret, err := kc.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   docker.SecretDir,
				EnableCache: false,
				ScratchDir:  docker.ScratchDir,
			}
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository failed")
			}

			// write secret and config in a temp dir
			// cleanup whole tempDir dir at the end
			tempDir, err := ioutil.TempDir("", "stash-cli")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			// prepare local dirs
			if err = localDirs.prepareSecretDir(tempDir, secret); err != nil {
				return err
			}
			if err = localDirs.prepareConfigDir(tempDir, &setupOpt, &restoreOpt); err != nil {
				return err
			}
			if err = localDirs.prepareDownloadDir(); err != nil {
				return err
			}

			// run restore inside docker
			if err = runRestoreViaDocker(*localDirs); err != nil {
				return err
			}
			log.Infof("Repository %s/%s restored in path %s", namespace, repositoryName, restoreOpt.Destination)
			return nil
		},
	}

	cmd.Flags().StringVar(&localDirs.downloadDir, "destination", localDirs.downloadDir, "Destination path where snapshot will be restored.")

	cmd.Flags().StringVar(&restoreOpt.SourceHost, "host", restoreOpt.SourceHost, "Name of the source host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestoreDirs, "directories", restoreOpt.RestoreDirs, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry for restic cli")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Restic docker image tag")

	return cmd
}

// FixIt! directly call restic/restic to restore hosts in (parallel ?)
func runRestoreViaDocker(localDirs cliLocalDirectories) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", localDirs.configDir + ":" + docker.ConfigDir,
		"-v", localDirs.secretDir + ":" + docker.SecretDir,
		"-v", localDirs.downloadDir + ":" + docker.DestinationDir,
		imgRestic.ToContainerImage(),
		"docker",
		"download-snapshots",
	}
	log.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}