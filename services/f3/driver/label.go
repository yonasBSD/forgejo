// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"

	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
	"lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

type Label struct {
	issues_model.Label
}

func LabelConverter(f *issues_model.Label) *Label {
	return &Label{
		Label: *f,
	}
}

func (o Label) GetID() int64 {
	return o.ID
}

func (o Label) GetName() string {
	return o.Name
}

func (o *Label) SetID(id int64) {
	o.ID = id
}

func (o *Label) IsNil() bool {
	return o.ID == 0
}

func (o *Label) Equals(other *Label) bool {
	return o.Name == other.Name
}

func (o *Label) ToFormat() *format.Label {
	return &format.Label{
		Common:      format.NewCommon(o.ID),
		Name:        o.Name,
		Color:       o.Color,
		Description: o.Description,
	}
}

func (o *Label) FromFormat(label *format.Label) {
	*o = Label{
		Label: issues_model.Label{
			ID:          label.GetID(),
			Name:        label.Name,
			Description: label.Description,
			Color:       label.Color,
		},
	}
}

type LabelProvider struct {
	g       *Forgejo
	project *ProjectProvider
}

func (o *LabelProvider) ToFormat(ctx context.Context, label *Label) *format.Label {
	return label.ToFormat()
}

func (o *LabelProvider) FromFormat(ctx context.Context, m *format.Label) *Label {
	var label Label
	label.FromFormat(m)
	return &label
}

func (o *LabelProvider) GetObjects(ctx context.Context, user *User, project *Project, page int) []*Label {
	labels, err := issues_model.GetLabelsByRepoID(ctx, project.GetID(), "", db.ListOptions{Page: page, PageSize: o.g.perPage})
	if err != nil {
		panic(fmt.Errorf("error while listing labels: %v", err))
	}

	r := util.ConvertMap[*issues_model.Label, *Label](labels, LabelConverter)
	if o.project != nil {
		o.project.labels = util.NewNameIDMap[*Label](r)
	}
	return r
}

func (o *LabelProvider) ProcessObject(ctx context.Context, user *User, project *Project, label *Label) {
}

func (o *LabelProvider) Get(ctx context.Context, user *User, project *Project, exemplar *Label) *Label {
	id := exemplar.GetID()
	label, err := issues_model.GetLabelInRepoByID(ctx, project.GetID(), id)
	if issues_model.IsErrRepoLabelNotExist(err) {
		return &Label{}
	}
	if err != nil {
		panic(err)
	}
	return LabelConverter(label)
}

func (o *LabelProvider) Put(ctx context.Context, user *User, project *Project, label *Label) *Label {
	l := label.Label
	l.RepoID = project.GetID()
	if err := issues_model.NewLabel(ctx, &l); err != nil {
		panic(err)
	}
	return o.Get(ctx, user, project, LabelConverter(&l))
}

func (o *LabelProvider) Delete(ctx context.Context, user *User, project *Project, label *Label) *Label {
	l := o.Get(ctx, user, project, label)
	if !l.IsNil() {
		if err := issues_model.DeleteLabel(project.GetID(), l.GetID()); err != nil {
			panic(err)
		}
	}
	return l
}
