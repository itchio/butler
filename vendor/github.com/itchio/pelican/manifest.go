package pelican

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

type node = map[string]interface{}

func visit(n node, key string, f func(c node)) {
	if c, ok := n[key].(node); ok {
		f(c)
	}
}

func visitMany(n node, key string, f func(c node)) {
	if cs, ok := n[key].([]node); ok {
		for _, c := range cs {
			f(c)
		}
	}
	if c, ok := n[key].(node); ok {
		f(c)
	}
}

func getString(n node, key string, f func(s string)) {
	if s, ok := n[key].(string); ok {
		f(s)
	}
}

func interpretManifest(info *PeInfo, manifest []byte) error {
	intermediate := make(node)
	err := json.Unmarshal([]byte(manifest), &intermediate)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	assInfo := &AssemblyInfo{}

	interpretIdentity := func(id node, f func(id *AssemblyIdentity)) {
		ai := &AssemblyIdentity{}
		getString(id, "-name", func(s string) { ai.Name = s })
		getString(id, "-version", func(s string) { ai.Version = s })
		getString(id, "-type", func(s string) { ai.Type = s })

		getString(id, "-processorArchitecture", func(s string) { ai.ProcessorArchitecture = s })
		getString(id, "-publicKeyToken", func(s string) { ai.PublicKeyToken = s })
		getString(id, "-language", func(s string) { ai.Language = s })
		f(ai)
	}

	visit(intermediate, "assembly", func(assembly node) {
		visit(assembly, "assemblyIdentity", func(id node) {
			interpretIdentity(id, func(ai *AssemblyIdentity) {
				assInfo.Identity = ai
			})
		})

		getString(assembly, "description", func(s string) { assInfo.Description = s })

		visit(assembly, "trustInfo", func(ti node) {
			visit(ti, "security", func(sec node) {
				visit(sec, "requestedPrivileges", func(rp node) {
					visit(rp, "requestedExecutionLevel", func(rel node) {
						getString(rel, "-level", func(s string) {
							assInfo.RequestedExecutionLevel = s
						})
					})
				})
			})
		})

		visit(assembly, "dependency", func(dep node) {
			visitMany(dep, "dependentAssembly", func(da node) {
				visit(da, "assemblyIdentity", func(id node) {
					interpretIdentity(id, func(ai *AssemblyIdentity) {
						info.DependentAssemblies = append(info.DependentAssemblies, ai)
					})
				})
			})
		})
	})

	info.AssemblyInfo = assInfo

	return nil
}
