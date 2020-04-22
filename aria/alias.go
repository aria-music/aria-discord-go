package aria

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type alias struct {
	Alias map[string][]string

	revMap map[string]string
}

func newAlias() (*alias, error) {
	file := "alias.json"

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open alias file: %w", err)
	}
	defer f.Close()

	a := new(alias)
	if err := json.NewDecoder(f).Decode(a); err != nil {
		return nil, fmt.Errorf("failed to decode alias.json: %w", err)
	}

	a.construct()
	return a, nil
}

func (a *alias) construct() {
	a.revMap = make(map[string]string)
	for k, v := range a.Alias {
		for _, al := range v {
			if c, ok := a.revMap[al]; ok {
				log.Printf("alias collision: %s -> %s, %s\n", al, c, k)
				continue
			}

			a.revMap[al] = k
		}
	}
}

func (a *alias) resolve(al string) string {
	return a.revMap[al]
}
