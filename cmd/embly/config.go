package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"embly/pkg/comms"
	localbuild "embly/pkg/local-build"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	validator "gopkg.in/go-playground/validator.v9"
)

type flags struct {
	command          string
	verbose          bool
	help             bool
	debug            bool
	emblyProjectFile string
}

type emblyBinContext struct {
	flags            *flags
	project          emblyProject
	functionRegistry map[string]projectFunction
	master           *comms.Master
	projectRoot      string
}

type emblyProject struct {
	Functions []projectFunction `json:"functions"`
	Gateways  []projectGateway  `json:"gateways"`
}

type projectFunction struct {
	Name    string `json:"name" validate:"required"`
	Path    string `json:"path" validate:"required"`
	Context string `json:"context"`
	Runtime string `json:"runtime" validate:"required"`
	module  string
}

type projectGateway struct {
	Name     string `json:"name" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	Function string `json:"function" validate:"required"`
}

func (pg *projectGateway) checkFunction(emblyCtx *emblyBinContext) (err error) {
	_, ok := emblyCtx.functionRegistry[pg.Function]
	if !ok {
		err = errors.Errorf("Function '%s' for gateway '%s' not found", pg.Function, pg.Name)
	}
	return
}

func (pg *projectGateway) getFunction(emblyCtx *emblyBinContext) projectFunction {
	return emblyCtx.functionRegistry[pg.Function]
}

func main() {
	if err := runMain(); err != nil {
		fmt.Println("Error running embly: ", err)
		os.Exit(1)
	}
}

func runMain() (err error) {
	emblyCtx := emblyBinContext{}
	emblyCtx.master = comms.NewMaster()
	emblyCtx.parseFlags()
	switch emblyCtx.flags.command {
	case "start":
		if err = emblyCtx.getEmblyProjectFile(); err != nil {
			return
		}
		if err = emblyCtx.buildFunctions(); err != nil {
			return
		}
		if err = emblyCtx.startGateways(); err != nil {
			return
		}
	default:
		emblyCtx.printUsage()
	}
	return nil
}

var f *flags

func (emblyCtx *emblyBinContext) parseFlags() {
	emblyCtx.flags = &flags{}
	f = emblyCtx.flags

	flag.BoolVar(&emblyCtx.flags.verbose, "v", false, "enable verbose logging")
	flag.BoolVar(&emblyCtx.flags.help, "h", false, "pring this message")
	flag.BoolVar(&emblyCtx.flags.debug, "d", false, "print stdout and stderr from wasm")
	flag.Parse()
	args := flag.Args()
	if len(args) != 0 {
		emblyCtx.flags.command = args[0]
	}
	return
}

func (emblyCtx *emblyBinContext) printUsage() {
	fmt.Printf("Embly\n\n")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nCommands:")
	fmt.Println("  start\trun this thing")
}

func (emblyCtx *emblyBinContext) getEmblyProjectFile() (err error) {
	var f *os.File
	if f, emblyCtx.projectRoot, err = findConfigFile(); err != nil {
		return
	}
	var ep *emblyProject
	if ep, err = parseConfigFile(f); err != nil {
		return
	}
	err = validateEmblyProjectFile(ep)
	emblyCtx.project = *ep
	return
}

func (emblyCtx *emblyBinContext) emblyBuildDir() string {
	return filepath.Join(emblyCtx.projectRoot, "embly_build")
}
func (emblyCtx *emblyBinContext) buildFunctions() (err error) {
	emblyBuildDir := emblyCtx.emblyBuildDir()
	ebFileInfo, _ := os.Stat(emblyBuildDir)
	if ebFileInfo == nil {
		if err = os.Mkdir(emblyBuildDir, os.ModePerm); err != nil {
			return err
		}
	} else {
		if !ebFileInfo.IsDir() {
			return errors.New("embly_build exists but it is not a directory")
		}
	}

	emblyCtx.functionRegistry = make(map[string]projectFunction)
	for i, fn := range emblyCtx.project.Functions {
		fmt.Println("building function with name", fn.Name)
		buildContext := filepath.Join(emblyCtx.projectRoot, fn.Context)
		buildLocation := filepath.Join(buildContext, fn.Path)
		wasmLocation := filepath.Join(emblyBuildDir, fn.Name+".out")
		if err = localbuild.Create(fn.Name, buildLocation, buildContext, wasmLocation); err != nil {
			return
		}
		fn.module = wasmLocation
		emblyCtx.project.Functions[i] = fn
		emblyCtx.functionRegistry[fn.Name] = fn
		fmt.Println(emblyCtx.functionRegistry)
	}
	return nil
}

type contextdata struct {
	contextID int
}

// TODO. this should might be the content of our wire messages between function and master
type functionMessage struct {
	data []byte
	id   int
	fn   string
}

type funcContextData struct {
	commGroup CommGroup
	commMap   map[int]int
	ebc       *emblyBinContext
}

func (emblyCtx *emblyBinContext) spawnInstance(name string, id int, commGroup CommGroup) {
	fmt.Println("spawning instance with name", name, id)
}

func (emblyCtx *emblyBinContext) launchHTTPGateway(g projectGateway) (err error) {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", g.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("request", r)

			masterG := emblyCtx.master.NewGateway()
			fn := g.getFunction(emblyCtx)
			masterFn := emblyCtx.master.NewFunction(fn.module, masterG.ID)
			masterG.AttachFn(masterFn)
			if err := masterFn.Start(); err != nil {
				log.Fatal(err)
			}
			// out, err := proxy.DumpRequest(r)

			// resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(b)), r)
			// if err != nil {
			// 	w.WriteHeader(500)
			// 	w.Write([]byte(err.Error()))
			// }
			// w.WriteHeader(resp.StatusCode)
			// for k, vs := range resp.Header {
			// 	for _, v := range vs {
			// 		w.Header().Add(k, v)
			// 	}
			// }
			// io.Copy(w, resp.Body)
		}),
	}
	fmt.Printf("HTTP gateway '%s' listening on port %d\n", g.Name, g.Port)
	go server.ListenAndServe()
	return nil
}

func (emblyCtx *emblyBinContext) handleTCPConn(conn net.Conn, g projectGateway) (err error) {
	fmt.Println("new tcp conn for", g.Name)

	// TODO: so fragile!
	masterG := emblyCtx.master.NewGateway()
	fn := g.getFunction(emblyCtx)
	masterFn := emblyCtx.master.NewFunction(fn.module, masterG.ID)
	masterG.AttachFn(masterFn)

	if err := masterFn.Start(); err != nil {
		log.Fatal(err)
	}
	go io.Copy(conn, masterG)
	io.Copy(masterG, conn)
	// m.NewFunction()
	return nil
}

func (emblyCtx *emblyBinContext) launchTCPGateway(g projectGateway) (err error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", g.Port))
	if err != nil {
		return err
	}
	fmt.Printf("TCP gateway '%s' listening on port %d\n", g.Name, g.Port)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Got error accepting on ", g.Name, err)
			}
			go emblyCtx.handleTCPConn(conn, g)
		}
	}()
	return nil
}

func (emblyCtx *emblyBinContext) startGateways() (err error) {
	go emblyCtx.master.Start()
	var wg sync.WaitGroup
	wg.Add(1)
	for _, g := range emblyCtx.project.Gateways {
		if err = g.checkFunction(emblyCtx); err != nil {
			return
		}
		switch kind := g.Type; kind {
		case "http":
			if err := emblyCtx.launchHTTPGateway(g); err != nil {
				return err
			}
		case "tcp":
			if err := emblyCtx.launchTCPGateway(g); err != nil {
				return err
			}
		default:
			return errors.Errorf("gateway type of '%s' not available", kind)
		}
	}
	wg.Wait()
	return
}

func parseConfigFile(f *os.File) (ep *emblyProject, err error) {
	var b []byte
	if b, err = ioutil.ReadAll(f); err != nil {
		return
	}
	ep = &emblyProject{}
	err = yaml.Unmarshal(b, ep)
	return
}

func validateEmblyProjectFile(ep *emblyProject) (err error) {
	if ep.Functions == nil {
		return errors.New("no functions in embly-project.yml file")
	}
	return validator.New().Struct(ep)
}

func findConfigFile() (f *os.File, location string, err error) {
	var wd string
	if wd, err = os.Getwd(); err != nil {
		return
	}
	for {
		if f, err = os.Open(filepath.Join(wd, "./embly-project.yml")); err == nil {
			break
		}
		parent := filepath.Join(wd, "../")
		if wd == parent || wd == "/" {
			break
		}
		wd = parent
	}
	location = wd
	if f == nil {
		err = errors.New("embly-project.yml not found in this directory or any parent")
		return
	}
	return
}