package circleci

import (
	"flag"
	"os"
)

func (this *CircleCi) BindFlags() {
	flag.StringVar(&this.User, "circle_user", "", "Circle user")
	flag.StringVar(&this.Project, "circle_project", "", "Circle project")
	flag.StringVar(&this.ApiToken, "circle_token", "", "Circle token")
	flag.Int64Var(&this.BuildNumber, "circle_buildnum", 0, "Circle build number")
	flag.StringVar(&this.TargetDir, "build_artifact_dir", os.Getenv("PWD"), "Target directory")
	flag.StringVar(&this.AuthZkPath, "circle_auth_zkpath", "", "Circle Auth zk path")
}
