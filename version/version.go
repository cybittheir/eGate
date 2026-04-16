package version

var (
	Name    = "EMail Gateway Proxy"
	Version = "0.2.0"
	Build   = "46"
)

func String() string {
	return Name + " v." + Version + " build " + Build
}
