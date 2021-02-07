package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"plugin"

	"github.com/aquasecurity/tracee/tracee-rules/signatures/rego/regosig"
	"github.com/aquasecurity/tracee/tracee-rules/types"
)

func getSignatures(rulesDir string, rules []string) ([]types.Signature, error) {
	if rulesDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			log.Print(err)
		}
		rulesDir = filepath.Join(filepath.Dir(exePath), "rules")
	}
	gosigs, err := findGoSigs(rulesDir)
	if err != nil {
		return nil, err
	}
	opasigs, err := findRegoSigs(rulesDir)
	if err != nil {
		return nil, err
	}
	sigs := append(gosigs, opasigs...)
	res := []types.Signature{}
	if rules == nil {
		res = sigs
	} else {
		for _, s := range sigs {
			for _, r := range rules {
				if m, err := s.GetMetadata(); err == nil && m.Name == r {
					res = append(res, s)
				}
			}
		}
	}
	return res, nil
}

func findGoSigs(dir string) ([]types.Signature, error) {
	var res []types.Signature
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error opening plugin %s: %v", info.Name(), err)
		}
		if filepath.Ext(info.Name()) != ".so" || info.isDir() == true {
			return filepath.SkipDir
		}
		p, err := plugin.Open(filepath.Join(path, info.Name()))
		if err != nil {
			log.Printf("error opening plugin %s: %v", info.Name(), err)
			return filepath.SkipDir
		}
		export, err := p.Lookup("ExportedSignatures")
		if err != nil {
			log.Printf("missing Export symbol in plugin %s", info.Name())
			return filepath.SkipDir
		}
		sigs := *export.(*[]types.Signature)
		res = append(res, sigs...)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func findRegoSigs(dir string) ([]types.Signature, error) {
	var res []types.Signature
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(file.Name()) != ".rego" || info.isDir() == true {
			return filepath.SkipDir
		}
		regoCode, err := ioutil.ReadFile(filepath.Join(dir, info.Name()))
		if err != nil {
			log.Printf("error reading file %s/%s: %v", dir, file, err)
			return filepath.SkipDir
		}
		sig, err := regosig.NewRegoSignature(string(regoCode))
		if err != nil {
			log.Printf("error creating rego signature with: %s: %v ", regoCode[0:20], err)
			return filepath.SkipDir
		}
		res = append(res, sig)
	})
	return res, nil
}
