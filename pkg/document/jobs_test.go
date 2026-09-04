package document

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func TestGetJobRows(t *testing.T) {
	rows := getJobRows([]gitlab.Job{
		{Name: ".base", Hidden: true},
		{Name: "compile", Stage: "build", Description: "Builds it."},
		{Name: "deploy", Stage: "deploy", When: "manual", Needs: []string{"compile"}, Extends: []string{".base"}},
	})

	assert.Len(t, rows, 2) // hidden .base dropped
	assert.Equal(t, "compile", rows[0].Name)
	assert.Equal(t, "Builds it.", rows[0].Description)
	assert.Equal(t, "manual", rows[1].When)
	assert.Equal(t, []string{"compile"}, rows[1].Needs)
	assert.Equal(t, []string{".base"}, rows[1].Extends)
}
