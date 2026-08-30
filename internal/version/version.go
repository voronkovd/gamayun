package version

// Set at link time:
//
//	-X github.com/voronkovd/gamayun/internal/version.Version=v1.2.3
//	-X github.com/voronkovd/gamayun/internal/version.Repo=voronkovd/gamayun
var (
	Version = "dev"
	Repo    = ""
)
