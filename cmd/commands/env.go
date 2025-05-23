/*
 *    Copyright 2025 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package commands

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
	"github.com/version-fox/vfox/internal/toolset"
)

var Env = &cli.Command{
	Name: "env",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "shell",
			Aliases: []string{"s"},
			Usage:   "shell name",
		},
		&cli.BoolFlag{
			Name:    "cleanup",
			Aliases: []string{"c"},
			Usage:   "cleanup old temp files",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "output json format",
		},
		&cli.BoolFlag{
			Name:  "full",
			Usage: "output full env",
		},
	},
	Action:   envCmd,
	Category: CategorySDK,
}

func envCmd(ctx *cli.Context) error {
	if ctx.IsSet("json") {
		return outputJSON()
	} else if ctx.IsSet("cleanup") {
		return cleanTmp()
	} else if ctx.IsSet("full") {
		return envFlag(ctx, "full")
	} else {
		return envFlag(ctx, "cwd")
	}
}

func outputJSON() error {
	type SDKs map[string]map[string]*string
	data := struct {
		IsHookEnv bool     `json:"is_hook_env"`
		Paths     []string `json:"paths"`
		SDKs      SDKs     `json:"sdks"`
	}{
		IsHookEnv: env.IsHookEnv(),
		Paths:     []string{},
		SDKs:      make(SDKs),
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	tvs, err := toolset.NewMultiToolVersions([]string{
		manager.PathMeta.WorkingDirectory,
		manager.PathMeta.CurTmpPath,
		manager.PathMeta.HomePath,
	})
	if err != nil {
		return err
	}
	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			if keys, err := lookupSdk.EnvKeys(base.Version(version), base.OriginalLocation); err == nil {
				data.SDKs[lookupSdk.Plugin.Name] = keys.Variables
				data.Paths = append(data.Paths, keys.Paths.Slice()...)
				return true
			}
		}
		return false
	})
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func cleanTmp() error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	// Clean up the old temp files, before today.
	manager.CleanTmp()
	return nil
}

func envFlag(ctx *cli.Context, mode string) error {
	shellName := ctx.String("shell")
	if shellName == "" {
		return cli.Exit("shell name is required", 1)
	}
	s := shell.NewShell(shellName)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", shellName)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	var sdkEnvs internal.SdkEnvs
	var err error
	if mode == "full" {
		sdkEnvs, err = manager.SessionEnvKeys(internal.SessionEnvOptions{
			WithGlobalEnv: true,
		})
	} else {
		sdkEnvs, err = manager.SessionEnvKeys(internal.SessionEnvOptions{
			WithGlobalEnv: false,
		})
	}

	if err != nil {
		return err
	}

	if len(sdkEnvs) == 0 {
		return nil
	}

	exportEnvs := sdkEnvs.ToExportEnvs()

	exportStr := s.Export(exportEnvs)
	fmt.Println(exportStr)
	return nil
}
