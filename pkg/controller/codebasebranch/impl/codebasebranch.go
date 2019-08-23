package impl

import (
	"codebase-operator/models"
	edpv1alpha1 "codebase-operator/pkg/apis/edp/v1alpha1"
	"codebase-operator/pkg/jenkins"
	ClientSet "codebase-operator/pkg/openshift"
	"codebase-operator/pkg/settings"
	"context"
	"fmt"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type CodebaseBranchService struct {
	Client client.Client
}

func (service CodebaseBranchService) Create(cr *edpv1alpha1.CodebaseBranch) {
	if cr.Status.Status != models.StatusInit {
		log.Printf("Release %v for application %v is not in init status. Skipped", cr.Spec.BranchName,
			cr.Spec.CodebaseName)
		return
	}

	err := service.updateStatusFields(cr, edpv1alpha1.CodebaseBranchStatus{
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.AcceptCodebaseBranchRegistration,
		Result:          "success",
		Value:           "inactive",
	})

	clientSet := ClientSet.CreateOpenshiftClients()
	log.Println("Client set has been created")

	releaseJob := fmt.Sprintf("%v/job/Create-release-%v", cr.Spec.CodebaseName, cr.Spec.CodebaseName)
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", cr.Namespace)
	jenkinsToken, jenkinsUsername, err := settings.GetJenkinsCreds(*clientSet, cr.Namespace)
	if err != nil {
		log.Println(err)
		service.setFailedFields(cr, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}

	log.Printf("Started creating release %v for application %v...", cr.Spec.BranchName, cr.Spec.CodebaseName)

	jenkinsClient, err := jenkins.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		log.Println(err)
		service.setFailedFields(cr, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}

	err = jenkinsClient.TriggerReleaseJob(cr.Spec.BranchName, cr.Spec.FromCommit, cr.Spec.CodebaseName)
	if err != nil {
		log.Println(err)
		service.setFailedFields(cr, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}
	log.Println("Release job has been triggered")

	jobStatus, err := jenkinsClient.GetJobStatus(releaseJob, 10*time.Second, 50)
	if err != nil {
		log.Println(err)
		service.setFailedFields(cr, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}
	if jobStatus == "blue" {
		cr.Status = edpv1alpha1.CodebaseBranchStatus{
			LastTimeUpdated: time.Now(),
			Username:        "system",
			Action:          edpv1alpha1.JenkinsConfiguration,
			Result:          edpv1alpha1.Success,
			Value:           "active",
		}
		log.Printf("Release has been created. Status: %v", models.StatusFinished)
	} else {
		log.Printf("Failed to create release. Release job status is '%v'. CodebaseBranch status: %v",
			jobStatus, models.StatusFailed)
		service.setFailedFields(cr, edpv1alpha1.JenkinsConfiguration, "Release job was failed.")
		return
	}
}
func (service CodebaseBranchService) Update(cr *edpv1alpha1.CodebaseBranch) {

}

func (service CodebaseBranchService) Delete(cr *edpv1alpha1.CodebaseBranch) {

}

func (service CodebaseBranchService) updateStatusFields(obj *edpv1alpha1.CodebaseBranch,
	status edpv1alpha1.CodebaseBranchStatus) error {
	obj.Status = status
	err := service.Client.Status().Update(context.TODO(), obj)
	if err != nil {
		err = service.Client.Update(context.TODO(), obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (service CodebaseBranchService) setFailedFields(obj *edpv1alpha1.CodebaseBranch,
	action edpv1alpha1.ActionType, message string) {
	obj.Status = edpv1alpha1.CodebaseBranchStatus{
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}
