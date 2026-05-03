package version

var (
	Name    = "EMail Gateway Proxy"
	Version = "0.3.0"
	Build   = "48"
)

func String() string {
	return Name + " v." + Version + " build " + Build
}
