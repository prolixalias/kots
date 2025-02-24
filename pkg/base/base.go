package base

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type Base struct {
	Path            string
	Namespace       string
	Files           []BaseFile
	ErrorFiles      []BaseFile
	AdditionalFiles []BaseFile
	Bases           []Base
}

type BaseFile struct {
	Path    string
	Content []byte
	Error   error
}

type OverlySimpleGVK struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name        string                 `yaml:"name"`
	Namespace   string                 `yaml:"namespace"`
	Annotations map[string]interface{} `json:"annotations"`
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func GetGVKWithNameAndNs(content []byte, baseNS string) (string, OverlySimpleGVK) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", o
	}

	namespace := baseNS
	if o.Metadata.Namespace != "" {
		namespace = o.Metadata.Namespace
	}

	return fmt.Sprintf("%s-%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name, namespace), o
}

func (f *BaseFile) transpileHelmHooksToKotsHooks() error {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(f.Content, nil, nil)
	if err != nil {
		return nil // this isn't an error, it's just not a job witih a hook, that's certain
	}

	// we currently only support hooks on jobs
	if gvk.Group != "batch" || gvk.Version != "v1" || gvk.Kind != "Job" {
		return nil
	}

	job := obj.(*batchv1.Job)

	helmHookDeletePolicyAnnotation, ok := job.Annotations["helm.sh/hook-delete-policy"]
	if !ok {
		return nil
	}

	job.Annotations["kots.io/hook-delete-policy"] = helmHookDeletePolicyAnnotation

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(job, &b); err != nil {
		return errors.Wrap(err, "failed to encode job")
	}

	f.Content = b.Bytes()
	return nil
}

type ParseError struct {
	Err error
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

// ShouldBeIncludedInBaseKustomization attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseKustomization(excludeKotsKinds bool) (bool, error) {
	var m interface{}

	if err := yaml.Unmarshal(f.Content, &m); err != nil {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			return false, ParseError{Err: err}
		}
		return false, nil
	}

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught in previous unmarshal

	// check if this is a kubernetes document
	if o.APIVersion == "" || o.Kind == "" {
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			// ignore empty files and files with only comments
			if m == nil {
				return false, nil
			}
			return false, ParseError{Err: errors.New("not a kubernetes document")}
		}
		return false, nil
	}

	// Backup is never deployed. kots.io/exclude and kots.io/when are used to enable snapshots
	if excludeKotsKinds {
		if iskotsAPIVersionKind(o) {
			return false, nil
		}
	}

	exclude, err := isExcludedByAnnotation(o.Metadata.Annotations)
	return !exclude, errors.Wrapf(err, "failed to check if object %s, kind %s/%s is excluded by annotation", o.Metadata.Name, o.APIVersion, o.Kind)
}

func isExcludedByAnnotation(annotations map[string]interface{}) (bool, error) {
	if annotations == nil {
		return false, nil
	}

	if val, ok := annotations["kots.io/exclude"]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal, nil
		}

		if strVal, ok := val.(string); ok {
			boolVal, err := strconv.ParseBool(strVal)
			if err != nil {
				// should this be a ParseError?
				return false, errors.Errorf("unable to parse %s as bool in exclude annotation", strVal)
			}

			return boolVal, nil
		}

		// should this be a ParseError?
		return false, errors.Errorf("unexpected type in exclude annotation: %T", val)
	}

	if val, ok := annotations["kots.io/when"]; ok {
		if boolVal, ok := val.(bool); ok {
			return !boolVal, nil
		}

		if strVal, ok := val.(string); ok {
			boolVal, err := strconv.ParseBool(strVal)
			if err != nil {
				// should this be a ParseError?
				return false, errors.Errorf("unable to parse %s as bool in when annotation", strVal)
			}

			return !boolVal, nil
		}

		// should this be a ParseError?
		return false, errors.Errorf("unexpected type in when annotation: %T", val)
	}

	return false, nil
}

func (f BaseFile) IsKotsKind() (bool, error) {
	var m interface{}

	if err := yaml.Unmarshal(f.Content, &m); err != nil {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			return false, ParseError{Err: err}
		}
		return false, nil
	}

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught in previous unmarshal

	// check if this is a kubernetes document
	if o.APIVersion == "" || o.Kind == "" {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			// ignore empty files and files with only comments
			if m == nil {
				return false, nil
			}
			return false, ParseError{Err: errors.New("not a kubernetes document")}
		}
		return false, nil
	}

	return iskotsAPIVersionKind(o), nil
}

func iskotsAPIVersionKind(o OverlySimpleGVK) bool {
	if o.APIVersion == "velero.io/v1" && o.Kind == "Backup" {
		return true
	}
	if o.APIVersion == "kots.io/v1beta1" {
		return true
	}
	if o.APIVersion == "troubleshoot.sh/v1beta2" {
		return true
	}
	if o.APIVersion == "troubleshoot.replicated.com/v1beta1" {
		return true
	}
	// In addition to kotskinds, we exclude the application crd for now
	if o.APIVersion == "app.k8s.io/v1beta1" {
		return true
	}
	return false
}

func (b Base) ListErrorFiles() []BaseFile {
	files := append([]BaseFile{}, b.ErrorFiles...)
	for _, b := range b.Bases {
		files = append(files, PrependBaseFilesPath(b.ListErrorFiles(), b.Path)...)
	}
	return files
}

func PrependBaseFilesPath(files []BaseFile, prefix string) []BaseFile {
	if prefix == "" {
		return files
	}
	next := []BaseFile{}
	for _, file := range files {
		file.Path = path.Join(prefix, file.Path)
		next = append(next, file)
	}
	return next
}
