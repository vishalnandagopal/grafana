package apistore

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
)

// Called on create
func (s *Storage) prepareObjectForStorage(ctx context.Context, newObject runtime.Object) ([]byte, error) {
	user, err := identity.GetRequester(ctx)
	if err != nil {
		return nil, err
	}

	obj, err := utils.MetaAccessor(newObject)
	if err != nil {
		return nil, err
	}
	if obj.GetName() == "" && obj.GetGenerateName() == "" {
		return nil, storage.ErrResourceVersionSetOnCreate
	}
	if obj.GetResourceVersion() != "" {
		return nil, storage.ErrResourceVersionSetOnCreate
	}

	if obj.GetName() == "" {
		obj.SetName(names.SimpleNameGenerator.GenerateName(obj.GetGenerateName()))
	}
	obj.SetGenerateName("") // Clear the random name field
	obj.SetResourceVersion("")
	obj.SetSelfLink("")

	// Read+write will verify that origin format is accurate
	origin, err := obj.GetOriginInfo()
	if err != nil {
		return nil, err
	}
	obj.SetOriginInfo(origin)
	obj.SetUpdatedBy("")
	obj.SetUpdatedTimestamp(nil)
	obj.SetCreatedBy(user.GetUID())
	obj.SetResourceVersion("")

	var buf bytes.Buffer
	err = s.codec.Encode(newObject, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Called on update
func (s *Storage) prepareObjectForUpdate(ctx context.Context, updateObject runtime.Object, previousObject runtime.Object) ([]byte, error) {
	user, err := identity.GetRequester(ctx)
	if err != nil {
		return nil, err
	}

	obj, err := utils.MetaAccessor(updateObject)
	if err != nil {
		return nil, err
	}
	if obj.GetName() == "" {
		return nil, fmt.Errorf("updated object must have a name")
	}

	previous, err := utils.MetaAccessor(previousObject)
	if err != nil {
		return nil, err
	}

	if previous.GetUID() == "" {
		return nil, fmt.Errorf("object is missing UID")
	}

	obj.SetUID(previous.GetUID())
	obj.SetCreatedBy(previous.GetCreatedBy())
	obj.SetCreationTimestamp(previous.GetCreationTimestamp())
	obj.SetResourceVersion("")
	obj.SetName(previous.GetName())

	if obj.GetGenerateName() == "" {
		obj.SetGenerateName(obj.GetName())
	}

	// Read+write will verify that origin format is accurate
	origin, err := obj.GetOriginInfo()
	if err != nil {
		return nil, err
	}
	obj.SetOriginInfo(origin)
	obj.SetUpdatedBy(user.GetUID())
	obj.SetUpdatedTimestampMillis(time.Now().UnixMilli())

	var buf bytes.Buffer
	err = s.codec.Encode(updateObject, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
