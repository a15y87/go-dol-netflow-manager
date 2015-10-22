package main
import (
	"github.com/a15y87/go-ipset"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/takama/daemon"
	"net/http"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)


const (
	name        = "dol-netflow-manager"
	description = "DOL netflow manager"
	port = ":2300"
)

var dependencies = []string{"network.target"}

var stdlog, errlog *log.Logger

type Service struct {
	daemon.Daemon
}

type IPhashArgs struct {
    Name string
	IPaddr string
	Timeout int
}

type IPhashReply struct {
    Message string
}

type IpHash struct {
}

func (h *IpHash) Set (r *http.Request, args *IPhashArgs, reply *IPhashReply) error {

	stdlog.Println("Got:", fmt.Sprint(args))
	s, err := go_ipset.New(args.Name, "hash:ip", &go_ipset.Params{})
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
	err = s.Add(args.IPaddr, args.Timeout)
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
    reply.Message = "True"
    return nil
}

func (h *IpHash) Get (r *http.Request, args *IPhashArgs, reply *IPhashReply) error {
	stdlog.Println("Got:", fmt.Sprint(args))
	s, err := go_ipset.New(args.Name, "hash:ip", &go_ipset.Params{})
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
	res, err := s.Test(args.IPaddr)
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
	reply.Message = fmt.Sprint(res)
	return nil
}

func (h *IpHash) Del (r *http.Request, args *IPhashArgs, reply *IPhashReply) error {
	stdlog.Println("Del:", fmt.Sprint(args))
	s, err := go_ipset.New(args.Name, "hash:ip", &go_ipset.Params{})
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
	err = s.Del(args.IPaddr)
	if err != nil {
		reply.Message = fmt.Sprint(err)
		return err
	}
	reply.Message = "True"
	return nil
}

func (service *Service) Manage() (string, error) {

	usage := "Usage: myservice install | remove | start | stop | status"

	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)


	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(new(IpHash), "")

	http.Handle("/api", s)
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return "Possibly was a problem with the port binding", err
	}

	go http.Serve(listener, nil)



	for {
		select {

		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			stdlog.Println("Stoping listening on ", listener.Addr())
			listener.Close()
			if killSignal == os.Interrupt {
				return "Daemon was interruped by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}
	return usage, nil
}


func init() {

	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
	srv, err := daemon.New(name, description, dependencies...)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}