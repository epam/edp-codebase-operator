package chain

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type BranchesExistsError string

func (e BranchesExistsError) Error() string {
	return string(e)
}

type DropJenkinsFolders struct {
	k8sClient client.Client
}

func NewDropJenkinsFolders(k8sClient client.Client) *DropJenkinsFolders {
	return &DropJenkinsFolders{k8sClient: k8sClient}
}

func (h *DropJenkinsFolders) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if c.Spec.CiTool != util.CIJenkins {
		log.Info("Skipping jenkins folders deletion because ci tool is not jenkins")

		return nil
	}

	log.Info("Starting to delete related jenkins folders")

	var branchList codebaseApi.CodebaseBranchList
	if err := h.k8sClient.List(ctx, &branchList, &client.ListOptions{
		Namespace: c.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list codebase branches: %w", err)
	}

	for i := 0; i < len(branchList.Items); i++ {
		for j := 0; j < len(branchList.Items[i].OwnerReferences); j++ {
			owner := branchList.Items[i].OwnerReferences[j]
			if owner.Kind == c.Kind && owner.UID == c.UID {
				return BranchesExistsError("can not delete jenkins folder while any codebase branches exists")
			}
		}
	}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", util.CodebaseLabelKey, c.Name))
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %w", err)
	}

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	var jenkinsFolderList jenkinsApi.JenkinsFolderList
	if err := h.k8sClient.List(ctx, &jenkinsFolderList, options); err != nil {
		return fmt.Errorf("failed to list jenkins folders: %w", err)
	}

	for i := 0; i < len(jenkinsFolderList.Items); i++ {
		jf := jenkinsFolderList.Items[i]
		log.Info("Trying to delete jenkins folder", "folder name", jf.Name)

		if err := h.k8sClient.Delete(ctx, &jf); err != nil {
			return fmt.Errorf("failed to delete jenkins folder: %w", err)
		}
	}

	log.Info("Done deleting child jenkins folders")

	return nil
}
