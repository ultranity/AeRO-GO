package main

import (
	"AeRO/proxy/util"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/rs/zerolog/log"
)

type RoundRobin struct {
	sync.Mutex

	current int
	pool    []string
}

var tagBalancer = make(map[string]*RoundRobin)

func GetRobin(tag string) *RoundRobin {
	if rb, ok := tagBalancer[tag]; ok {
		return rb
	} else {
		rb = &RoundRobin{
			current: 0,
			pool:    []string{},
		}
		return rb
	}
}

// this method will return a string of addr server from list server.
func (r *RoundRobin) update(pool []string) string {
	r.Lock()
	defer r.Unlock()
	r.pool = pool
	if len(pool) == 0 {
		return ""
	}
	if r.current >= len(r.pool) {
		r.current %= len(r.pool)
	}

	result := r.pool[r.current]
	r.current++
	return result
}

func (server *Server) MuxServer(addr string, domain string) {
	if addr == "" {
		return
	}
	app := fiber.New()
	app.Use(compress.New())
	app.Get("/metrics", monitor.New())
	if domain == "" {
		app.Use("/:tag/:name/*", func(c *fiber.Ctx) error {
			tag, name := c.Params("tag"), c.Params("name")
			avail := util.Map(server.filterManagers([]string{tag}), func(mgr *Manager) string {
				return mgr.Proxies[name].Port
			})
			if len(avail) == 0 {
				c.SendStatus(404)
				return nil
			}

			selected := GetRobin(tag).update(avail)
			selected = c.Protocol() + "://localhost:" + selected
			target := strings.TrimPrefix(c.OriginalURL(), "/"+tag+"/"+name)
			c.Request().Header.Add("X-Real-IP", c.IP())
			if err := proxy.Do(c, selected+target); err != nil {
				return err
			}
			// Remove Server header from response
			c.Response().Header.Del(fiber.HeaderServer)
			return nil
		})
	} else {
		app.Use("/", func(c *fiber.Ctx) error {
			host := strings.TrimSuffix(c.Hostname(), domain)
			tagname := util.SplitX(host, ".")
			tag, name := tagname[0], tagname[1]
			avail := util.Map(server.filterManagers([]string{tag}), func(mgr *Manager) string {
				return mgr.Proxies[name].Port
			})
			if len(avail) == 0 {
				c.SendStatus(404)
				return nil
			}
			selected := c.Protocol() + "://localhost:" + GetRobin(tag).update(avail)
			target := c.OriginalURL()
			c.Request().Header.Add("X-Real-IP", c.IP())
			if err := proxy.Do(c, selected+target); err != nil {
				return err
			}
			// Remove Server header from response
			c.Response().Header.Del(fiber.HeaderServer)
			return nil
		})
	}
	app.Listen(addr)
	log.Info().Msgf("mux listen on %s", addr)
}
