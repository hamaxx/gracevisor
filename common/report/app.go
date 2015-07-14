package report

type App struct {
	Name string
	Host string
	Port uint16

	Instances []*Instance
}
