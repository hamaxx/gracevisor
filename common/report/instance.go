package report

type Instance struct {
	Id                uint32
	Active            bool
	Host              string
	Port              uint16
	Status            string
	SinceStatusChange uint64
	Error             string
}
