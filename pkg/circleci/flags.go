package circleci

import (
	"flag"
	"os"
)

func (this *CircleCi) BindFlags() {
	flag.StringVar(&this.User, "circle_user", "", "Circle user")
	flag.StringVar(&this.Project, "circle_project", "", "Circle project")
	flag.StringVar(&this.ApiToken, "circle_token", "", "Circle token")
	flag.StringVar(&this.ProjectUser, "circle_project_user", "", "Circle project user")
	flag.IntVar(&this.BuildNum, "circle_buildnum", 0, "Circle build number")
	flag.IntVar(&this.PreviousBuildNum, "circle_previous_buildnum", 0, "Circle previous build number")
	flag.StringVar(&this.GitRepo, "circle_git_repo", "", "Circle git repo")
	flag.StringVar(&this.GitBranch, "circle_git_branch", "", "Circle branch")
	flag.StringVar(&this.Commit, "circle_commit", "", "Circle commit")
	flag.StringVar(&this.ArtifactsDir, "build_artifact_dir", os.Getenv("PWD"), "Build artifacts directory")
	flag.StringVar(&this.AuthZkPath, "circle_auth_zkpath", "", "Circle Auth zk path")
}
