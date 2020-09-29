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
	directory   string
	extension   string
	development bool
	httpFileSys http.FileSystem
}

// Engine struct
type Engine struct {
	config Config
	loaded bool
	// lock for funcmap and templates
	mutex     sync.RWMutex
	functions map[string]interface{}
	Templates *jet.Set
}

// Init returns Engine struct
func Init(config ...Config) *Engine {
	if config[0].extension != ".jet" && config[0].extension != ".jet.html" && config[0].extension != ".html.jet" {
		log.Fatalf("Error: Config.extension must be one of these => (.jet, .jet.html, .html.jet)")
	}

	return &Engine{
		config:    config[0],
		functions: make(map[string]interface{}),
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

// Load the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Templates = jet.NewHTMLSetLoader(multi.NewLoader(
		jet.NewOSFileSystemLoader(e.config.directory),
		httpfs.NewLoader(e.config.httpFileSys),
	))
	for name, fn := range e.functions {
		e.Templates.AddGlobal(name, fn)
	}
	e.Templates.SetDevelopmentMode(e.config.development)
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
		return fmt.Errorf("render: template %s does not exist", template)
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
