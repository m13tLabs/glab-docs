package document

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func TestGetJobRows(t *testing.T) {
	rows := getJobRows([]gitlab.Job{
		{Name: ".base", Hidden: true},
		{Name: ".proxy_setup", Hidden: true, Description: "Sets up the proxy config."},
		{Name: "compile", Stage: "build", Description: "Builds it."},
		{Name: "deploy", Stage: "deploy", When: "manual", Needs: []string{"compile"}, Extends: []string{".base"}},
	})

	assert.Len(t, rows, 3) // undocumented hidden .base dropped, documented .proxy_setup kept
	assert.Equal(t, ".proxy_setup", rows[0].Name)
	assert.Equal(t, "Sets up the proxy config.", rows[0].Description)
	assert.True(t, rows[0].Hidden)
	assert.Equal(t, "compile", rows[1].Name)
	assert.Equal(t, "Builds it.", rows[1].Description)
	assert.Equal(t, "manual", rows[2].When)
	assert.Equal(t, []string{"compile"}, rows[2].Needs)
	assert.Equal(t, []string{".base"}, rows[2].Extends)
}
