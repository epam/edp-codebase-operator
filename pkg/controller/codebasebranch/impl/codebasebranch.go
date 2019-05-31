package impl

import (
	"codebase-operator/models"
	edpv1alpha1 "codebase-operator/pkg/apis/edp/v1alpha1"
	"codebase-operator/pkg/jenkins"
	ClientSet "codebase-operator/pkg/openshift"
	"codebase-operator/pkg/settings"
	"fmt"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type CodebaseBranchService struct {
	Client client.Client
}

func (codebaseBranch CodebaseBranchService) Create(cr *edpv1alpha1.CodebaseBranch) {
	if cr.Status.Status != models.StatusInit {
		log.Printf("Release %v for application %v is not in init status. Skipped", cr.Spec.BranchName,
			cr.Spec.CodebaseName)
		return
	}

	clientSet := ClientSet.CreateOpenshiftClients()
	log.Println("Client set has been created")
	releaseJob := fmt.Sprintf("%v/job/Create-release-%v", cr.Spec.CodebaseName, cr.Spec.CodebaseName)
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", cr.Namespace)
	jenkinsToken, jenkinsUsername, err := settings.GetJenkinsCreds(*clientSet, cr.Namespace)
	if err != nil {
		log.Println(err)
		rollback(cr)
		return
	}

	log.Printf("Started creating release %v for application %v...", cr.Spec.BranchName, cr.Spec.CodebaseName)

	jenkinsClient, err := jenkins.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		log.Println(err)
		rollback(cr)
		return
	}

	err = jenkinsClient.TriggerReleaseJob(cr.Spec.BranchName, cr.Spec.FromCommit, cr.Spec.CodebaseName)
	if err != nil {
		log.Println(err)
		rollback(cr)
		return
	}
	log.Println("Release job has been triggered")

	jobStatus, err := jenkinsClient.GetJobStatus(releaseJob, 10*time.Second, 50)
	if err != nil {
		log.Println(err)
		rollback(cr)
		return
	}
	if jobStatus == "blue" {
		setStatusFields(cr, models.StatusFinished, time.Now())
		log.Printf("Release has been created. Status: %v", models.StatusFinished)
	} else {
		log.Printf("Failed to create release. Release job status is '%v'. CodebaseBranch status: %v",
			jobStatus, models.StatusFailed)
		rollback(cr)
		return
	}
}

func rollback(cr *edpv1alpha1.CodebaseBranch) {
	setStatusFields(cr, models.StatusFailed, time.Now())
}

func setStatusFields(cr *edpv1alpha1.CodebaseBranch, status string, time time.Time) {
	cr.Status.Status = status
	cr.Status.LastTimeUpdated = time
	cr.Status.Username = "system"
	log.Printf("Status for application release %v has been updated to '%v' at %v.", cr.Spec.BranchName, status, time)
}

func (codebaseBranch CodebaseBranchService) Update(cr *edpv1alpha1.CodebaseBranch) {

}

func (codebaseBranch CodebaseBranchService) Delete(cr *edpv1alpha1.CodebaseBranch) {

}
