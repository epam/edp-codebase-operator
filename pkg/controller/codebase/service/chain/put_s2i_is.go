package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/platform"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
)

type PutS2iIs struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutS2iIs) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("start creating s2i is...")
	if err := h.tryToSetupS2I(*c); err != nil {
		setFailedFields(c, v1alpha1.PutS2I, err.Error())
		return err
	}
	rLog.Info("end creating s2i is...")
	return nextServeOrNil(h.next, c)
}

func (h PutS2iIs) tryToSetupS2I(c v1alpha1.Codebase) error {
	log.Info("start creating image stream", "codebase name", c.Name)
	if c.Spec.Lang == util.LanguageJava {
		log.V(2).Info("skip creating s2i for java lang", "name", c.Name)
		return nil
	}

	if !isSupportedType(c) {
		log.Info("couldn't create image stream as type of codebase is not acceptable")
		return nil
	}

	if c.Spec.Strategy == util.ImportStrategy && platform.IsK8S() {
		return nil
	}

	is, err := util.GetAppImageStream(c.Spec.Lang)
	if err != nil {
		return err
	}
	return util.CreateS2IImageStream(*h.clientSet.ImageClient, c.Name, c.Namespace, is)
}

func isSupportedType(c v1alpha1.Codebase) bool {
	return c.Spec.Type == util.Application && c.Spec.Lang != util.OtherLanguage
}
