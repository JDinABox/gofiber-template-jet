package jet

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/CloudyKit/jet/v5"
	"github.com/CloudyKit/jet/v5/loaders/httpfs"
	"github.com/CloudyKit/jet/v5/loaders/multi"
	"github.com/gofiber/fiber/v2"
)

// Config struct
type Config struct {
	Directory   string
	Extension   string
	Development bool
	HTTPFileSys http.FileSystem
}

// Engine struct
type Engine struct {
	config Config
	loaded bool
	// lock for funcmap and templates
	mutex     sync.RWMutex
	functions map[string]interface{}
	globals   map[string]string
	Templates *jet.Set
}

// Init returns Engine struct
func Init(config ...Config) *Engine {
	if config[0].Extension != ".jet" && config[0].Extension != ".jet.html" && config[0].Extension != ".html.jet" {
		log.Fatalf("Error: Config.extension must be one of these => (.jet, .jet.html, .html.jet)")
	}

	return &Engine{
		config:    config[0],
		functions: make(map[string]interface{}),
		globals:   make(map[string]string)
	}
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
// From: gofiber/template
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.mutex.Lock()
	e.functions[name] = fn
	e.mutex.Unlock()
	return e
}

// AddGlobal adds Global to the template's Global map
func (e *Engine) AddGlobal(name string, value string) *Engine {
	e.mutex.Lock()
	e.globals[name] = value
	e.mutex.Unlock()
	return e
}

// Load the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Templates = jet.NewHTMLSetLoader(multi.NewLoader(
		jet.NewOSFileSystemLoader(e.config.Directory),
		httpfs.NewLoader(e.config.HTTPFileSys),
	))

	for name, fn := range e.functions {
		e.Templates.AddGlobal(name, fn)
	}
	for name, value := range e.globals {
		e.Templates.AddGlobal(name, value)
	}

	e.Templates.SetDevelopmentMode(e.config.Development)
	e.loaded = true

	return nil
}

// Render the templates
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if !e.loaded {
		if err := e.Load(); err != nil {
			return err
		}
	}

	tmpl, err := e.Templates.GetTemplate(template)
	if err != nil || tmpl == nil {
		return fmt.Errorf("render: template %s does not exist \n %s", template, err)
	}

	bind := jetVarMap(binding)

	return tmpl.Execute(out, bind, nil)
}

// From: gofiber/template
func jetVarMap(binding interface{}) jet.VarMap {
	var bind jet.VarMap
	if binding == nil {
		return bind
	}
	if binds, ok := binding.(map[string]interface{}); ok {
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	} else if binds, ok := binding.(fiber.Map); ok {
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	} else if binds, ok := binding.(jet.VarMap); ok {
		bind = binds
	}
	return bind
}
