package util

import "testing"

func TestParseGitProjectPath(t *testing.T) {
	cases := map[string]string{
		"git@gitlab.example.com:infra/jobs/gitlab-components/helpers.git":            "infra/jobs/gitlab-components/helpers",
		"git@gitlab.example.com:infra/jobs/gitlab-components/helpers":                "infra/jobs/gitlab-components/helpers",
		"https://gitlab.example.com/infra/jobs/gitlab-components/helpers.git":        "infra/jobs/gitlab-components/helpers",
		"https://user@gitlab.example.com/infra/jobs/gitlab-components/helpers.git":   "infra/jobs/gitlab-components/helpers",
		"ssh://git@gitlab.example.com:2222/infra/jobs/gitlab-components/helpers.git": "infra/jobs/gitlab-components/helpers",
		"":     "",
		"junk": "",
	}

	for remote, want := range cases {
		if got := parseGitProjectPath(remote); got != want {
			t.Errorf("parseGitProjectPath(%q) = %q, want %q", remote, got, want)
		}
	}
}
