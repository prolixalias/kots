package handlers

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type MetadataResponse struct {
	IconURI       string `json:"iconUri"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	IsKurlEnabled bool   `json:"isKurlEnabled"`
	UpstreamURI   string `json:"upstreamUri"`
}

// Metadata route is UNAUTHENTICATED
// It is needed for branding/some cluster flags before user is logged in.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	// This is not an authenticated request

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	isKurlEnabled := kurl.IsKurl()

	brandingConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	metadataResponse := MetadataResponse{
		IconURI:       "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png",
		Name:          "the application",
		Namespace:     util.PodNamespace,
		IsKurlEnabled: isKurlEnabled,
	}

	if err == nil {
		data, ok := brandingConfigMap.Data["application.yaml"]
		if !ok {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		// parse data as a kotskind
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(data), nil, nil)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
			logger.Error(errors.New("unexpected gvk found in metadata"))
			w.WriteHeader(500)
			return
		}

		application := obj.(*kotsv1beta1.Application)
		metadataResponse.IconURI = application.Spec.Icon
		metadataResponse.Name = application.Spec.Title
		metadataResponse.UpstreamURI = brandingConfigMap.Data["upstreamUri"]
	}

	JSON(w, 200, metadataResponse)
}
