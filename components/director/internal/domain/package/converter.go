package mp_package

import (
	"database/sql"
	"encoding/json"

	"github.com/kyma-incubator/compass/components/director/pkg/apperrors"

	"github.com/kyma-incubator/compass/components/director/pkg/str"

	"github.com/kyma-incubator/compass/components/director/internal/model"
	"github.com/kyma-incubator/compass/components/director/internal/repo"
	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	"github.com/pkg/errors"
)

//go:generate mockery -name=AuthConverter -output=automock -outpkg=automock -case=underscore
type AuthConverter interface {
	ToGraphQL(in *model.Auth) (*graphql.Auth, error)
	InputFromGraphQL(in *graphql.AuthInput) (*model.AuthInput, error)
}

type converter struct {
	auth     AuthConverter
	api      APIConverter
	event    EventConverter
	document DocumentConverter
}

func NewConverter(auth AuthConverter, api APIConverter, event EventConverter, document DocumentConverter) *converter {
	return &converter{
		auth:     auth,
		api:      api,
		event:    event,
		document: document,
	}
}

func (c *converter) ToEntity(in *model.Package) (*Entity, error) {
	if in == nil {
		return nil, nil
	}

	defaultInstanceAuth, err := c.marshalDefaultInstanceAuth(in.DefaultInstanceAuth)
	if err != nil {
		return nil, err
	}

	output := &Entity{
		ID:                            in.ID,
		TenantID:                      in.TenantID,
		ApplicationID:                 in.ApplicationID,
		Name:                          in.Name,
		Description:                   repo.NewNullableString(in.Description),
		DefaultInstanceAuth:           repo.NewNullableString(defaultInstanceAuth),
		InstanceAuthRequestJSONSchema: repo.NewNullableString(in.InstanceAuthRequestInputSchema),
	}

	return output, nil
}

func (c *converter) FromEntity(entity *Entity) (*model.Package, error) {
	if entity == nil {
		return nil, apperrors.NewInternalError("the Package entity is nil")
	}

	defaultInstanceAuth, err := c.unmarshalDefaultInstanceAuth(entity.DefaultInstanceAuth)
	if err != nil {
		return nil, err
	}

	output := &model.Package{
		ID:                             entity.ID,
		TenantID:                       entity.TenantID,
		ApplicationID:                  entity.ApplicationID,
		Name:                           entity.Name,
		Description:                    repo.StringPtrFromNullableString(entity.Description),
		DefaultInstanceAuth:            defaultInstanceAuth,
		InstanceAuthRequestInputSchema: repo.StringPtrFromNullableString(entity.InstanceAuthRequestJSONSchema),
	}

	return output, nil
}

func (c *converter) ToGraphQL(in *model.Package) (*graphql.Package, error) {
	if in == nil {
		return nil, apperrors.NewInternalError("the model Package is nil")
	}

	auth, err := c.auth.ToGraphQL(in.DefaultInstanceAuth)
	if err != nil {
		return nil, errors.Wrap(err, "while converting DefaultInstanceAuth to GraphQL")
	}

	return &graphql.Package{
		ID:                             in.ID,
		Name:                           in.Name,
		Description:                    in.Description,
		InstanceAuthRequestInputSchema: c.strPtrToJSONSchemaPtr(in.InstanceAuthRequestInputSchema),
		DefaultInstanceAuth:            auth,
	}, nil
}

func (c *converter) MultipleToGraphQL(in []*model.Package) ([]*graphql.Package, error) {
	var packages []*graphql.Package
	for _, r := range in {
		if r == nil {
			continue
		}
		pkg, err := c.ToGraphQL(r)
		if err != nil {
			return nil, errors.Wrap(err, "while converting Package to GraphQL")
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (c *converter) CreateInputFromGraphQL(in graphql.PackageCreateInput) (model.PackageCreateInput, error) {
	auth, err := c.auth.InputFromGraphQL(in.DefaultInstanceAuth)
	if err != nil {
		return model.PackageCreateInput{}, errors.Wrap(err, "while converting DefaultInstanceAuth input")
	}

	apiDefs, err := c.api.MultipleInputFromGraphQL(in.APIDefinitions)
	if err != nil {
		return model.PackageCreateInput{}, errors.Wrap(err, "while converting APIDefinitions input")
	}

	documents, err := c.document.MultipleInputFromGraphQL(in.Documents)
	if err != nil {
		return model.PackageCreateInput{}, errors.Wrap(err, "while converting Documents input")
	}

	eventDefs, err := c.event.MultipleInputFromGraphQL(in.EventDefinitions)
	if err != nil {
		return model.PackageCreateInput{}, errors.Wrap(err, "while converting EventDefinitions input")
	}

	return model.PackageCreateInput{
		Name:                           in.Name,
		Description:                    in.Description,
		InstanceAuthRequestInputSchema: c.jsonSchemaPtrToStrPtr(in.InstanceAuthRequestInputSchema),
		DefaultInstanceAuth:            auth,
		APIDefinitions:                 apiDefs,
		EventDefinitions:               eventDefs,
		Documents:                      documents,
	}, nil
}

func (c *converter) MultipleCreateInputFromGraphQL(in []*graphql.PackageCreateInput) ([]*model.PackageCreateInput, error) {
	var packages []*model.PackageCreateInput
	for _, item := range in {
		if item == nil {
			continue
		}
		pkg, err := c.CreateInputFromGraphQL(*item)
		if err != nil {
			return nil, err
		}
		packages = append(packages, &pkg)
	}

	return packages, nil
}

func (c *converter) UpdateInputFromGraphQL(in graphql.PackageUpdateInput) (*model.PackageUpdateInput, error) {
	auth, err := c.auth.InputFromGraphQL(in.DefaultInstanceAuth)
	if err != nil {
		return nil, errors.Wrap(err, "while converting DefaultInstanceAuth from GraphQL")
	}

	return &model.PackageUpdateInput{
		Name:                           in.Name,
		Description:                    in.Description,
		InstanceAuthRequestInputSchema: c.jsonSchemaPtrToStrPtr(in.InstanceAuthRequestInputSchema),
		DefaultInstanceAuth:            auth,
	}, nil
}

func (c *converter) marshalDefaultInstanceAuth(defaultInstanceAuth *model.Auth) (*string, error) {
	if defaultInstanceAuth == nil {
		return nil, nil
	}

	output, err := json.Marshal(defaultInstanceAuth)
	if err != nil {
		return nil, errors.Wrap(err, "while marshaling default auth")
	}
	return str.Ptr(string(output)), nil
}

func (c *converter) unmarshalDefaultInstanceAuth(defaultInstanceAuthSql sql.NullString) (*model.Auth, error) {
	var defaultInstanceAuth *model.Auth
	if defaultInstanceAuthSql.Valid && defaultInstanceAuthSql.String != "" {
		defaultInstanceAuth = &model.Auth{}
		err := json.Unmarshal([]byte(defaultInstanceAuthSql.String), defaultInstanceAuth)
		if err != nil {
			return nil, errors.Wrap(err, "while unmarshalling default instance auth")
		}
	}

	return defaultInstanceAuth, nil
}

func (c *converter) strPtrToJSONSchemaPtr(in *string) *graphql.JSONSchema {
	if in == nil {
		return nil
	}
	out := graphql.JSONSchema(*in)
	return &out
}

func (c *converter) jsonSchemaPtrToStrPtr(in *graphql.JSONSchema) *string {
	if in == nil {
		return nil
	}
	out := string(*in)
	return &out
}
