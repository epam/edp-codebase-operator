package util

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	imageV1 "github.com/openshift/api/image/v1"
	imageV1Client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetAppImageStream(lang string) (*imageV1.ImageStream, error) {
	log.Info("trying to get image stream", "is", lang)
	switch strings.ToLower(lang) {
	case model.JavaScript:
		return newS2IReact(lang), nil
	case model.Java:
		return newS2IJava(lang), nil
	case model.DotNet:
		return newS2IDotNet(lang), nil
	case model.GroovyPipeline:
		return newS2IGroovyPipeline(lang), nil
	}
	return nil, nil
}

func CreateS2IImageStream(c imageV1Client.ImageV1Client, codebaseName, namespace string, is *imageV1.ImageStream) error {
	log.Info("trying to create s2i image stream", "codebase name", codebaseName, "namespace", namespace)
	_, err := c.
		ImageStreams(namespace).
		Get(is.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		_, err := c.
			ImageStreams(namespace).
			Create(is)
		if err != nil {
			return err
		}
		log.Info("image stream in Openshift has been created", "codebase name", codebaseName)
	}
	log.Info("Image stream in Openshift already exist. Creation skipped", "codebase name", codebaseName)
	return nil
}

func newS2IReact(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + strings.ToLower(lang),
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name:        "latest",
				Annotations: nil,
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "epamedp/s2i-nginx:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IJava(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + strings.ToLower(lang),
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name:        "latest",
				Annotations: nil,
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "epamedp/s2i-java:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IDotNet(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "s2i-" + strings.ToLower(lang),
			Annotations: map[string]string{"openshift.io/display-name": ".NET Core Builder Images"},
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name: "latest",
				Annotations: map[string]string{
					"description": "Build and run .NET Core 2.0 applications on CentOS 7. For more " +
						"information about using this builder image, including OpenShift considerations, " +
						"see https://github.com/redhat-developer/s2i-dotnetcore/tree/master/2.0/build/README.md.",
					"iconClass":                 "icon-dotnet",
					"openshift.io/display-name": ".NET Core 2.0",
					"sampleContextDir":          "app",
					"sampleRef":                 "dotnetcore-2.0",
					"sampleRepo":                "https://github.com/redhat-developer/s2i-dotnetcore-ex.git",
					"supports":                  "dotnet:2.0,dotnet",
					"tags":                      "builder,.net,dotnet,dotnetcore,rh-dotnet20",
					"version":                   "2.0",
				},
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "epamedp/dotnet-20-centos7:latest",
				},
				ImportPolicy: imageV1.TagImportPolicy{},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IGroovyPipeline(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + strings.ToLower(lang),
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name:        "latest",
				Annotations: nil,
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "epamedp/s2i-java:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}
