/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"io"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/pkg/constants"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/target"
	"sigs.k8s.io/kustomize/pkg/transformerconfig"
)

type buildOptions struct {
	kustomizationPath      string
	outputPath             string
	transformerconfigPaths []string
}

var examples = `
Use the file somedir/kustomization.yaml to generate a set of api resources:
    build somedir

Use a url pointing to a remote directory/kustomization.yaml to generate a set of api resources:
    build url
The url should follow hashicorp/go-getter URL format described in
https://github.com/hashicorp/go-getter#url-format

url examples:
  sigs.k8s.io/kustomize//examples/multibases?ref=v1.0.6
  github.com/Liujingfang1/mysql
  github.com/Liujingfang1/kustomize//examples/helloWorld?ref=repoUrl2

Advanced usage:
Use different transformer configurations by passing files to kustomize
    build somedir -t someconfigdir
    build somedir -t some-transformer-configfile,another-transformer-configfile
`

// newCmdBuild creates a new build command.
func newCmdBuild(out io.Writer, fs fs.FileSystem) *cobra.Command {
	var o buildOptions
	var p string

	cmd := &cobra.Command{
		Use:          "build [path]",
		Short:        "Print current configuration per contents of " + constants.KustomizationFileName,
		Example:      examples,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate(args, p, fs)
			if err != nil {
				return err
			}
			return o.RunBuild(out, fs)
		},
	}
	cmd.Flags().StringVarP(
		&o.outputPath,
		"output", "o", "",
		"If specified, write the build output to this path.")
	cmd.Flags().StringVarP(
		&p,
		"transformer-config", "t", "",
		"If specified, use the transformer configs load from these files.")
	return cmd
}

// Validate validates build command.
func (o *buildOptions) Validate(args []string, p string, fs fs.FileSystem) error {
	if len(args) > 1 {
		return errors.New("specify one path to " + constants.KustomizationFileName)
	}
	if len(args) == 0 {
		o.kustomizationPath = "./"
		return nil
	}
	o.kustomizationPath = args[0]

	if p == "" {
		return nil
	}

	if fs.IsDir(p) {
		paths, err := fs.Glob(p + "/*")
		if err != nil {
			return err
		}
		o.transformerconfigPaths = paths
	} else {
		o.transformerconfigPaths = strings.Split(p, ",")
	}

	return nil
}

// RunBuild runs build command.
func (o *buildOptions) RunBuild(out io.Writer, fSys fs.FileSystem) error {
	rootLoader, err := loader.NewLoader(o.kustomizationPath, "", fSys)
	if err != nil {
		return err
	}
	defer rootLoader.Cleanup()
	target, err := target.NewKustTarget(
		rootLoader, fSys, makeTransformerconfig(fSys, o.transformerconfigPaths))
	if err != nil {
		return err
	}
	allResources, err := target.MakeCustomizedResMap()
	if err != nil {
		return err
	}
	// Output the objects.
	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return err
	}
	if o.outputPath != "" {
		return fSys.WriteFile(o.outputPath, res)
	}
	_, err = out.Write(res)
	return err
}

// makeTransformerConfig returns a complete TransformerConfig object from either files
// or the default configs
func makeTransformerconfig(fSys fs.FileSystem, paths []string) *transformerconfig.TransformerConfig {
	ldr, err := loader.NewLoader(".", "", fSys)
	if err != nil {
		log.Fatalf("error creating loader to load transformer configurations %v\n", err)
	}
	if paths == nil || len(paths) == 0 {
		return transformerconfig.MakeDefaultTransformerConfig()
	}
	t, err := transformerconfig.MakeTransformerConfigFromFiles(ldr, paths)
	if err != nil {
		log.Fatalf("failed to load transformer configrations %v\n", err)
	}
	return t
}
