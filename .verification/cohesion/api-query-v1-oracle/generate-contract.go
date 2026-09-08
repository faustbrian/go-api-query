// Command generate-contract emits the deterministic per-record digest contract
// for one bounded oracle output on standard output.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const maximumOutputBytes = 8 << 20

func main() {
	if len(os.Args) != 2 {
		panic("usage: generate-contract OUTPUT")
	}
	data, err := readFile(os.Args[1])
	mustContract(err)
	var output struct {
		Schema  string `json:"schema"`
		Records []struct {
			Name  string          `json:"name"`
			Value json.RawMessage `json:"value"`
		} `json:"records"`
	}
	mustContract(decodeClosed(data, &output))
	if output.Schema != "api-query-v1-oracle-output-v1" || len(output.Records) > 256 {
		panic("unexpected oracle output")
	}
	contract := struct {
		Schema  string `json:"schema"`
		Records []struct {
			Name        string `json:"name"`
			ValueSHA256 string `json:"value_sha256"`
		} `json:"records"`
	}{Schema: "api-query-v1-oracle-output-contract-v1"}
	contract.Records = make([]struct {
		Name        string `json:"name"`
		ValueSHA256 string `json:"value_sha256"`
	}, len(output.Records))
	seen := make(map[string]struct{}, len(output.Records))
	for index, item := range output.Records {
		if item.Name == "" {
			panic("empty record name")
		}
		if _, duplicate := seen[item.Name]; duplicate {
			panic("duplicate record name")
		}
		seen[item.Name] = struct{}{}
		contract.Records[index].Name = item.Name
		contract.Records[index].ValueSHA256 = canonicalDigest(item.Value)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	mustContract(encoder.Encode(contract))
}

func canonicalDigest(raw json.RawMessage) string {
	var value any
	mustContract(json.Unmarshal(raw, &value))
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	mustContract(encoder.Encode(value))
	digest := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(digest[:])
}

func readFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximumOutputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maximumOutputBytes {
		return nil, errors.New("oracle output exceeds maximum bytes")
	}
	return data, nil
}

func decodeClosed(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

func mustContract(err error) {
	if err != nil {
		panic(err)
	}
}
