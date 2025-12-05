package protocolfactory

type ServiceFactory struct {}

func New() *ServiceFactory {
    return &ServiceFactory{}
}
