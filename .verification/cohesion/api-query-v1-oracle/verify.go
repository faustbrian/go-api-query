// Command verify validates the closed API Query v1 oracle evidence packet.
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	modulePath          = "github.com/faustbrian/go-api-query"
	release             = "v1.0.0"
	releaseCommit       = "b69acc7edf26bafbca4496132715b2b71cc753b2"
	oracleRoot          = ".verification/cohesion/api-query-v1-oracle"
	sharedPath          = oracleRoot + "/shared-test-input.json"
	maxEvidenceBytes    = 8 << 20
	maxBinaryBytes      = 128 << 20
	maxJSONDepth        = 64
	maxArtifactCount    = 64
	maxCoverageItems    = 16
	maxCheckpointCount  = 32
	maxOracleRecords    = 256
	maxModuleZipEntries = 4096
	maxModuleZipBytes   = 64 << 20
	expectedInputSHA    = "7287d53d7afffbe8f158081f20cc6dffe4a2d06da771285036d6b467e6bc67d8"
	expectedHarnessSHA  = "08ef180f1531be0b038b40d9adf6a69489890ed45b11d4987b4f976001d8153e"
	expectedOutputSHA   = "9da16adc97f2074c3e890a290a8b7028938803fdc702d4f2689cfd8c9cd3816b"
	expectedContractSHA = "29265bc97313b2905e4858ce127471e1e24b58ece720b1e26cb7bc86ca48c516"
)

type artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type releaseArtifact struct {
	Path   string `json:"path"`
	Blob   string `json:"blob"`
	Bytes  int    `json:"bytes,omitempty"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	Schema         string   `json:"schema"`
	Module         string   `json:"module"`
	Decision       artifact `json:"decision"`
	DecisionFreeze artifact `json:"decision_freeze"`
	Release        struct {
		Version        string            `json:"version"`
		TagKind        string            `json:"tag_kind"`
		TagObject      string            `json:"tag_object"`
		PeeledCommit   string            `json:"peeled_commit"`
		Tree           string            `json:"tree"`
		ModuleSum      string            `json:"module_sum"`
		GoModSum       string            `json:"go_mod_sum"`
		AllowedSigners artifact          `json:"allowed_signers"`
		APIBaseline    releaseArtifact   `json:"api_baseline"`
		LegacySources  []releaseArtifact `json:"legacy_sources"`
	} `json:"release"`
	Artifacts []artifact `json:"artifacts"`
	Execution struct {
		Input   artifact `json:"input"`
		Result  artifact `json:"result"`
		Outcome string   `json:"outcome"`
	} `json:"execution"`
	Coverage []string `json:"coverage"`
}

type sharedInput struct {
	Schema string `json:"schema"`
	Source struct {
		Module          string   `json:"module"`
		Version         string   `json:"version"`
		ModuleSum       string   `json:"module_sum"`
		GoModSum        string   `json:"go_mod_sum"`
		ModuleZipSHA256 string   `json:"module_zip_sha256"`
		DownloadReceipt artifact `json:"download_receipt"`
	} `json:"source"`
	DependencyGraph artifact `json:"dependency_graph"`
	ExternalModule  struct {
		GoMod               artifact `json:"go_mod"`
		GoSum               artifact `json:"go_sum"`
		ReplaceDirectives   int      `json:"replace_directives"`
		WorkspaceDirectives int      `json:"workspace_directives"`
	} `json:"external_module"`
	Toolchain struct {
		Identity         string `json:"identity"`
		ExecutableSHA256 string `json:"executable_sha256"`
	} `json:"toolchain"`
	PublicResolution struct {
		Command   string `json:"command"`
		Outcome   string `json:"outcome"`
		GOPROXY   string `json:"goproxy"`
		GOSUMDB   string `json:"gosumdb"`
		GONOSUMDB string `json:"gonosumdb"`
		GOPRIVATE string `json:"goprivate"`
	} `json:"public_resolution"`
	Environment struct {
		GOOS, GOARCH, CGOEnabled, GOFLAGS, GOWORK string
		GOCACHE, GOMODCACHE, GOTMPDIR, TMPDIR     string
	} `json:"-"`
	RawEnvironment struct {
		GOOS       string `json:"goos"`
		GOARCH     string `json:"goarch"`
		CGOEnabled string `json:"cgo_enabled"`
		GOFLAGS    string `json:"goflags"`
		GOWORK     string `json:"gowork"`
		GOCACHE    string `json:"gocache"`
		GOMODCACHE string `json:"gomodcache"`
		GOTMPDIR   string `json:"gotmpdir"`
		TMPDIR     string `json:"tmpdir"`
	} `json:"environment"`
	Postgres struct {
		RepositoryDigest string   `json:"repository_digest"`
		ImageID          string   `json:"image_id"`
		ImageInspect     artifact `json:"image_inspect"`
		ContainerID      string   `json:"container_id"`
		ContainerInspect artifact `json:"container_inspect"`
		Database         string   `json:"database"`
		User             string   `json:"user"`
		DSN              string   `json:"dsn"`
	} `json:"postgres"`
	CommandContract struct {
		Oracle            string `json:"oracle"`
		PackageTestPrefix string `json:"package_test_prefix"`
	} `json:"command_contract"`
}

type executionResult struct {
	Schema      string   `json:"schema"`
	SharedInput artifact `json:"shared_input"`
	Oracle      struct {
		Command  string   `json:"command"`
		Source   artifact `json:"source"`
		Input    artifact `json:"input"`
		Outcome  string   `json:"outcome"`
		Output   artifact `json:"output"`
		Contract artifact `json:"contract"`
	} `json:"oracle"`
	PublishedPackageTests struct {
		Outcome                     string              `json:"outcome"`
		PackageCount                int                 `json:"package_count"`
		Passed                      int                 `json:"passed"`
		Failed                      int                 `json:"failed"`
		PostgresIntegrationExecuted bool                `json:"postgres_integration_executed"`
		Checkpoints                 []checkpointSummary `json:"checkpoints"`
	} `json:"published_package_tests"`
}

type checkpointSummary struct {
	Path                   string `json:"path"`
	SHA256                 string `json:"sha256"`
	Package                string `json:"package"`
	Command                string `json:"command"`
	InputFingerprintSHA256 string `json:"input_fingerprint_sha256"`
	Outcome                string `json:"outcome"`
}

type checkpoint struct {
	Schema            string   `json:"schema"`
	Package           string   `json:"package"`
	Command           string   `json:"command"`
	ExecutionRevision string   `json:"execution_revision"`
	SharedInput       artifact `json:"shared_input"`
	InputFingerprint  struct {
		Algorithm string `json:"algorithm"`
		SHA256    string `json:"sha256"`
	} `json:"input_fingerprint"`
	Result struct {
		Outcome string   `json:"outcome"`
		Log     artifact `json:"log"`
	} `json:"result"`
}

type oracleInput struct {
	Schema string `json:"schema"`
	HTTP   []struct {
		Name     string `json:"name"`
		Raw      string `json:"raw"`
		MaxBytes int    `json:"max_bytes"`
	} `json:"http"`
	PostgresOperators []string `json:"postgres_operators"`
}

type oracleOutput struct {
	Schema  string `json:"schema"`
	Records []struct {
		Name  string          `json:"name"`
		Value json.RawMessage `json:"value"`
	} `json:"records"`
}

type outputContract struct {
	Schema  string `json:"schema"`
	Records []struct {
		Name        string `json:"name"`
		ValueSHA256 string `json:"value_sha256"`
	} `json:"records"`
}

type downloadReceipt struct {
	Path, Version, Sum, GoModSum string
	Origin                       struct {
		VCS, URL, Hash, Ref string
	}
}

type dockerContainer struct {
	ID     string `json:"Id"`
	Image  string `json:"Image"`
	Config struct {
		Image string   `json:"Image"`
		Env   []string `json:"Env"`
	} `json:"Config"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

type dockerImage struct {
	ID           string   `json:"Id"`
	RepoDigests  []string `json:"RepoDigests"`
	Architecture string   `json:"Architecture"`
	OS           string   `json:"Os"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: verify MANIFEST")
	}
	manifestPath, err := filepath.Abs(os.Args[1])
	must(err)
	manifestPath, err = filepath.EvalSymlinks(manifestPath)
	must(err)
	repositoryRoot, err := filepath.EvalSymlinks(filepath.Clean(filepath.Join(filepath.Dir(manifestPath), "..", "..", "..")))
	must(err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repositoryTop, err := filepath.EvalSymlinks(gitOutput(ctx, repositoryRoot, "rev-parse", "--show-toplevel"))
	must(err)
	require(repositoryTop == repositoryRoot, "repository root identity")
	require(manifestPath == localPath(repositoryRoot, oracleRoot+"/manifest.json"), "canonical manifest path")
	var packet manifest
	must(decodeClosedFile(manifestPath, &packet))
	validateManifest(ctx, repositoryRoot, packet)
	fmt.Println("api-query v1 oracle packet is valid")
}

func validateManifest(ctx context.Context, root string, packet manifest) {
	require(packet.Schema == "api-query-v1-oracle-manifest-v2" && packet.Module == modulePath, "manifest identity")
	require(len(packet.Artifacts) <= maxArtifactCount && len(packet.Coverage) <= maxCoverageItems, "manifest collection bounds")
	require(packet.Decision.Path == oracleRoot+"/decision.md" && packet.Decision.SHA256 == "2b966f479b4953a2b3457c518110e5280f6a10ec267ae468eb472ed1bf08bca5", "decision identity")
	validateArtifact(root, packet.Decision)
	require(packet.DecisionFreeze.Path == oracleRoot+"/decision-freeze.json" && packet.DecisionFreeze.SHA256 == "500758defec6f4747e55469f380278fda3a27a8b1d8a7dbb84c7b0ec28ac9feb", "decision freeze identity")
	validateArtifact(root, packet.DecisionFreeze)
	require(packet.Release.Version == release && packet.Release.TagKind == "annotated", "release identity")
	require(packet.Release.TagObject == "ba85d7bc2a4d18e5f3a1b224c1d342a84b4c9b20" && packet.Release.PeeledCommit == releaseCommit && packet.Release.Tree == "9a8c5f253649757afbf13f42b0c7c05171d8972a", "release objects")
	validateReleaseObjects(ctx, root, packet)
	require(packet.Release.ModuleSum == "h1:rXR1zo1cbIETTx5y+Rp+zyDp0fMWphYIz8gKTwEQETg=" && packet.Release.GoModSum == "h1:mPb20YI2uLgkJ8DGdfAYmWa1zSH+dZt6I9AP2P4Er2s=", "release module sums")
	require(packet.Release.AllowedSigners.Path == oracleRoot+"/release-allowed-signers" && packet.Release.AllowedSigners.SHA256 == "333c9ce49cbc4f726189cc0d86adba997e3a8b36d80dca2e2d48cbcc91a83f96", "release signer identity")
	validateArtifact(root, packet.Release.AllowedSigners)
	validateReleaseArtifact(ctx, root, releaseCommit, packet.Release.APIBaseline, "ec28983aeaa00eedbfa61d2a6c5eeb67fc6c3185", 25021, "b8eac531590599de50da2ed042591db7edf07c6a438e900085aca53ed6354fc5")
	expectedSources := map[string]string{
		"apiqueryhttp/http.go":             "255f760cec857433c4162de8d56351f545a7f4f4",
		"apiqueryjsonapi/jsonapi.go":       "95e925c60cde7c77aecd9e4223badd73a83c182e",
		"apiquerypgx/pgx.go":               "0d5e99de7cba55f7a47c4ad07ac2f150b9a9e271",
		"apiqueryrpc/rpc.go":               "e481cbf1e37d888b98694eb6f46f5b8880566ca7",
		"apiqueryvalidation/validation.go": "04d7a29f84fb32cb84b45bccc6dfa2b1b2673edb",
	}
	require(len(packet.Release.LegacySources) == len(expectedSources), "legacy source cardinality")
	for _, source := range packet.Release.LegacySources {
		blob, ok := expectedSources[source.Path]
		require(ok, "legacy source path")
		validateReleaseArtifact(ctx, root, releaseCommit, source, blob, 0, source.SHA256)
	}
	validateArtifactSet(root, packet.Artifacts)
	require(packet.Execution.Outcome == "passed" && packet.Execution.Input.Path == sharedPath, "execution identity")
	validateArtifact(root, packet.Execution.Input)
	validateArtifact(root, packet.Execution.Result)
	validateExecution(ctx, root, packet)
	expectedCoverage := []string{
		"HTTP success, invalid classes, exact byte limit, and sentinel equality",
		"JSON:API family presence, callback order, nil and panic outcomes, and copy isolation",
		"PostgreSQL mapping copy, all operators, constraint precedence, fragments, arguments, and real outcomes",
		"JSON-RPC strict decode, bounds, empty and absent forms, request conversion, and nested descriptor copy",
		"Validation nil, structured and bounded generic errors, nested paths, and immutable returned values",
		"released source, API baseline, dependency, module graph, toolchain, and execution-input binding",
		"observed stateless execution, goroutine stability, file-descriptor stability, and no returned resource owners",
	}
	require(equalStrings(packet.Coverage, expectedCoverage), "coverage contract")
}

func validateExecution(ctx context.Context, root string, packet manifest) {
	var input sharedInput
	must(decodeClosedFile(localPath(root, packet.Execution.Input.Path), &input))
	require(input.Schema == "api-query-v1-package-test-shared-input-v1", "shared input schema")
	require(input.Source.Module == modulePath && input.Source.Version == release && input.Source.ModuleSum == packet.Release.ModuleSum && input.Source.GoModSum == packet.Release.GoModSum, "shared source identity")
	moduleZip := localPath(root, oracleRoot+"/published-module.zip")
	require(input.Source.ModuleZipSHA256 == digestFile(moduleZip), "published module zip")
	moduleSum, err := moduleZipSum(moduleZip)
	must(err)
	require(moduleSum == input.Source.ModuleSum, "published module canonical checksum")
	validateArtifact(root, input.Source.DownloadReceipt)
	validateArtifact(root, input.DependencyGraph)
	validateArtifact(root, input.ExternalModule.GoMod)
	validateArtifact(root, input.ExternalModule.GoSum)
	require(input.ExternalModule.ReplaceDirectives == 0 && input.ExternalModule.WorkspaceDirectives == 0, "external module isolation")
	goMod, err := readBoundedFile(localPath(root, input.ExternalModule.GoMod.Path), maxEvidenceBytes)
	must(err)
	validateExternalModule(ctx, root, input, goMod)
	validateDownloadReceipt(root, input)
	validateToolchain(input)
	resolution := input.PublicResolution
	require(resolution.Command == "go mod download -json github.com/faustbrian/go-api-query@v1.0.0" && resolution.Outcome == "passed", "public module download command")
	require(resolution.GOPROXY == "https://proxy.golang.org" && resolution.GOSUMDB == "sum.golang.org" && resolution.GONOSUMDB == "" && resolution.GOPRIVATE == "", "public module trust configuration")
	env := input.RawEnvironment
	require(env.GOOS == "darwin" && env.GOARCH == "arm64" && env.CGOEnabled == "1" && env.GOFLAGS == "" && env.GOWORK == "off", "execution environment")
	require(env.GOCACHE == "task-owned-disposable" && env.GOMODCACHE == "task-owned-disposable" && env.GOTMPDIR == "task-owned-disposable" && env.TMPDIR == "task-owned-disposable", "disposable environment")
	validatePostgres(root, input)
	require(input.CommandContract.Oracle == "go run ./harness.go ./input.json" && input.CommandContract.PackageTestPrefix == "go test -count=1 -json", "command contract")

	var result executionResult
	must(decodeClosedFile(localPath(root, packet.Execution.Result.Path), &result))
	require(result.Schema == "api-query-v1-oracle-execution-result-v2", "execution result schema")
	require(result.SharedInput == packet.Execution.Input, "execution input binding")
	require(result.Oracle.Command == input.CommandContract.Oracle && result.Oracle.Outcome == "passed", "oracle execution")
	require(result.Oracle.Source == (artifact{Path: oracleRoot + "/harness.go", SHA256: expectedHarnessSHA}), "oracle source identity")
	require(result.Oracle.Input == (artifact{Path: oracleRoot + "/input.json", SHA256: expectedInputSHA}), "oracle input identity")
	require(result.Oracle.Output == (artifact{Path: oracleRoot + "/output.json", SHA256: expectedOutputSHA}), "oracle output identity")
	require(result.Oracle.Contract == (artifact{Path: oracleRoot + "/output-contract.json", SHA256: expectedContractSHA}), "oracle contract identity")
	validateArtifact(root, result.Oracle.Source)
	validateArtifact(root, result.Oracle.Input)
	validateArtifact(root, result.Oracle.Output)
	validateArtifact(root, result.Oracle.Contract)
	validateOutput(root, result.Oracle.Output, result.Oracle.Contract)
	validateCheckpoints(root, input, result.PublishedPackageTests)
}

func validateDownloadReceipt(root string, input sharedInput) {
	var receipt downloadReceipt
	must(decodeClosedFile(localPath(root, input.Source.DownloadReceipt.Path), &receipt))
	require(receipt.Path == modulePath && receipt.Version == release && receipt.Sum == input.Source.ModuleSum && receipt.GoModSum == input.Source.GoModSum, "download receipt module")
	require(receipt.Origin.VCS == "git" && receipt.Origin.URL == "https://github.com/faustbrian/go-api-query" && receipt.Origin.Hash == releaseCommit && receipt.Origin.Ref == "refs/tags/v1.0.0", "download receipt origin")
}

func validateToolchain(input sharedInput) {
	require(input.Toolchain.Identity == "go version go1.26.6 darwin/arm64" && input.Toolchain.ExecutableSHA256 == "a1c83801d1756c3eca78366c6b585f2c21c20694fb1c7eb92c446a0580420412", "toolchain receipt")
}

func validateExternalModule(ctx context.Context, root string, input sharedInput, goMod []byte) {
	require(digestBytes(goMod) == "2112850e1bb1ca072fe8a7c0b9b3a87e1add3a9d1d666127046e655ab7d7bf97", "external go.mod identity")
	require(digestFile(localPath(root, input.ExternalModule.GoSum.Path)) == "4746caa1433eba170248f21207f427ca0df6d20ed04960af7d3ff5ff67a88b70", "external go.sum identity")
	command := exec.CommandContext(ctx, "go", "mod", "edit", "-json", "-modfile="+localPath(root, input.ExternalModule.GoMod.Path))
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.Output()
	must(err)
	var parsed struct {
		Module  struct{ Path string }
		Go      string
		Require []struct {
			Path    string
			Version string
		}
		Replace []json.RawMessage
	}
	must(json.Unmarshal(output, &parsed))
	require(parsed.Module.Path == "oracle.invalid/api-query-v1" && parsed.Go == "1.26.6" && len(parsed.Replace) == 0, "external go.mod structure")
	require(digestFile(localPath(root, input.DependencyGraph.Path)) == "0aee4c263acc2aa052bc56df4e169cf95f73557d00303212b064640ebd17ed31", "executed dependency graph identity")
}

func validatePostgres(root string, input sharedInput) {
	postgres := input.Postgres
	require(postgres.RepositoryDigest == "postgres@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15" && postgres.ImageID == "sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15", "PostgreSQL image identity")
	validateArtifact(root, postgres.ImageInspect)
	validateArtifact(root, postgres.ContainerInspect)
	var images []dockerImage
	must(decodeClosedFile(localPath(root, postgres.ImageInspect.Path), &images))
	require(len(images) == 1 && images[0].ID == postgres.ImageID && len(images[0].RepoDigests) == 1 && images[0].RepoDigests[0] == postgres.RepositoryDigest && images[0].Architecture == "arm64" && images[0].OS == "linux", "image inspect binding")
	var containers []dockerContainer
	must(decodeClosedFile(localPath(root, postgres.ContainerInspect.Path), &containers))
	require(len(containers) == 1 && containers[0].ID == postgres.ContainerID && containers[0].Image == postgres.ImageID && containers[0].Config.Image == postgres.RepositoryDigest && containers[0].State.Status == "running", "container inspect binding")
	expectedEnvironment := []string{"POSTGRES_PASSWORD=<redacted>", "POSTGRES_DB=" + postgres.Database, "POSTGRES_USER=" + postgres.User}
	require(equalStrings(containers[0].Config.Env, expectedEnvironment), "container environment projection")
	parsed, err := url.Parse(postgres.DSN)
	must(err)
	require(len(containers[0].NetworkSettings.Ports) == 1, "container port projection")
	bindings := containers[0].NetworkSettings.Ports["5432/tcp"]
	password, hasPassword := parsed.User.Password()
	port, portErr := strconv.Atoi(parsed.Port())
	require(portErr == nil && port > 0 && port <= 65535, "DSN port bound")
	require(parsed.Scheme == "postgres" && parsed.Hostname() == "127.0.0.1" && len(bindings) == 1 && parsed.Port() == bindings[0].HostPort && parsed.EscapedPath() == "/"+postgres.Database && parsed.User.Username() == postgres.User && hasPassword && password == "<redacted>" && parsed.RawQuery == "sslmode=disable" && parsed.Fragment == "" && parsed.Opaque == "", "DSN container binding")
}

func validateCheckpoints(root string, input sharedInput, tests struct {
	Outcome                     string              `json:"outcome"`
	PackageCount                int                 `json:"package_count"`
	Passed                      int                 `json:"passed"`
	Failed                      int                 `json:"failed"`
	PostgresIntegrationExecuted bool                `json:"postgres_integration_executed"`
	Checkpoints                 []checkpointSummary `json:"checkpoints"`
}) {
	expected := []string{modulePath, modulePath + "/apiqueryhttp", modulePath + "/apiqueryjsonapi", modulePath + "/apiquerypgx", modulePath + "/apiqueryrpc", modulePath + "/apiquerytest", modulePath + "/apiqueryvalidation", modulePath + "/cursor", modulePath + "/internal/strictjson"}
	expectedCheckpointPaths := []string{oracleRoot + "/checkpoints/root.json", oracleRoot + "/checkpoints/root__apiqueryhttp.json", oracleRoot + "/checkpoints/root__apiqueryjsonapi.json", oracleRoot + "/checkpoints/root__apiquerypgx.json", oracleRoot + "/checkpoints/root__apiqueryrpc.json", oracleRoot + "/checkpoints/root__apiquerytest.json", oracleRoot + "/checkpoints/root__apiqueryvalidation.json", oracleRoot + "/checkpoints/root__cursor.json", oracleRoot + "/checkpoints/root__internal__strictjson.json"}
	expectedLogPaths := []string{oracleRoot + "/test-logs/root.jsonl", oracleRoot + "/test-logs/root__apiqueryhttp.jsonl", oracleRoot + "/test-logs/root__apiqueryjsonapi.jsonl", oracleRoot + "/test-logs/root__apiquerypgx.jsonl", oracleRoot + "/test-logs/root__apiqueryrpc.jsonl", oracleRoot + "/test-logs/root__apiquerytest.jsonl", oracleRoot + "/test-logs/root__apiqueryvalidation.jsonl", oracleRoot + "/test-logs/root__cursor.jsonl", oracleRoot + "/test-logs/root__internal__strictjson.jsonl"}
	require(len(tests.Checkpoints) <= maxCheckpointCount, "checkpoint collection bound")
	require(tests.Outcome == "passed" && tests.PackageCount == 9 && tests.Passed == 9 && tests.Failed == 0 && tests.PostgresIntegrationExecuted && len(tests.Checkpoints) == 9, "package aggregate")
	for index, summary := range tests.Checkpoints {
		require(summary.Package == expected[index] && summary.Path == expectedCheckpointPaths[index] && summary.Outcome == "pass", "checkpoint summary order")
		validateArtifact(root, artifact{Path: summary.Path, SHA256: summary.SHA256})
		var item checkpoint
		must(decodeClosedFile(localPath(root, summary.Path), &item))
		require(item.Schema == "api-query-v1-package-test-checkpoint-v1" && item.Package == summary.Package && item.Command == summary.Command && item.ExecutionRevision == releaseCommit && item.SharedInput.Path == sharedPath && item.SharedInput.SHA256 == digestFile(localPath(root, sharedPath)), "checkpoint identity")
		require(item.InputFingerprint.Algorithm == "sha256(lines:schema,package,command,execution_revision,shared_input_sha256)", "fingerprint algorithm")
		payload := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", "api-query-v1-package-test-fingerprint-v1", item.Package, item.Command, item.ExecutionRevision, item.SharedInput.SHA256)
		require(digestBytes([]byte(payload)) == item.InputFingerprint.SHA256 && item.InputFingerprint.SHA256 == summary.InputFingerprintSHA256, "recomputed package fingerprint")
		require(item.Command == input.CommandContract.PackageTestPrefix+" "+item.Package && item.Result.Outcome == "pass", "checkpoint result")
		require(item.Result.Log.Path == expectedLogPaths[index], "checkpoint log path")
		validateArtifact(root, item.Result.Log)
		validateTestLog(root, item.Package, item.Result.Log.Path, item.Package == modulePath+"/apiquerypgx")
	}
}

func validateTestLog(root, packageName, logPath string, requirePostgres bool) {
	file, err := os.Open(localPath(root, logPath))
	must(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	packagePassed, postgresPassed := false, false
	for scanner.Scan() {
		var event struct {
			Time, Action, Package, Test, Output string
			Elapsed                             float64
		}
		must(json.Unmarshal(scanner.Bytes(), &event))
		if event.Package == packageName && event.Action == "pass" && event.Test == "" {
			packagePassed = true
		}
		if event.Package == packageName && event.Action == "pass" && event.Test == "TestPostgresInjectionResistanceAndStableCursorOrder" {
			postgresPassed = true
		}
	}
	must(scanner.Err())
	require(packagePassed && (!requirePostgres || postgresPassed), "test log outcome "+packageName)
}

func validateOutput(root string, outputArtifact, contractArtifact artifact) {
	var input oracleInput
	must(decodeClosedFile(localPath(root, ".verification/cohesion/api-query-v1-oracle/input.json"), &input))
	require(digestFile(localPath(root, oracleRoot+"/input.json")) == expectedInputSHA, "oracle canonical input bytes")
	require(input.Schema == "api-query-v1-oracle-input-v1" && len(input.HTTP) == 18 && len(input.PostgresOperators) == 16, "oracle input matrix")
	expectedHTTPNames := []string{"complete", "explicit-empty", "empty", "exact-byte-limit", "disabled-limit", "excess-bytes", "invalid-raw-utf8", "malformed-encoding", "semicolon", "unknown", "duplicate", "invalid-decoded-utf8", "empty-field-member", "empty-include-member", "invalid-filter", "empty-sort-name", "invalid-page-size", "invalid-page-offset"}
	seenHTTP := map[string]struct{}{}
	for index, item := range input.HTTP {
		require(item.Name == expectedHTTPNames[index] && item.MaxBytes >= 0 && item.MaxBytes <= 2048, "HTTP oracle input contract")
		_, duplicate := seenHTTP[item.Name]
		require(!duplicate, "duplicate HTTP oracle case")
		seenHTTP[item.Name] = struct{}{}
	}
	expectedOperators := []string{"eq", "neq", "lt", "lte", "gt", "gte", "in", "not_in", "between", "is_null", "contains", "starts_with", "ends_with", "and", "or", "not"}
	require(equalStrings(input.PostgresOperators, expectedOperators), "PostgreSQL operator matrix")
	var output oracleOutput
	must(decodeClosedFile(localPath(root, outputArtifact.Path), &output))
	var contract outputContract
	must(decodeClosedFile(localPath(root, contractArtifact.Path), &contract))
	require(len(output.Records) <= maxOracleRecords && len(contract.Records) <= maxOracleRecords, "output collection bounds")
	require(digestFile(localPath(root, outputArtifact.Path)) == expectedOutputSHA && digestFile(localPath(root, contractArtifact.Path)) == expectedContractSHA, "oracle canonical output bytes")
	require(output.Schema == "api-query-v1-oracle-output-v1" && contract.Schema == "api-query-v1-oracle-output-contract-v1" && len(output.Records) == 92 && len(contract.Records) == len(output.Records), "output contract")
	expectedRecordNames := []string{"input-schema", "sentinel-http", "sentinel-jsonapi-invalid", "sentinel-jsonapi-unsupported", "sentinel-postgres", "sentinel-jsonrpc", "named-types", "cursor-sentinel-invalid", "cursor-sentinel-expired", "cursor-sentinel-version", "cursor-sentinel-schema", "cursor-sentinel-sort", "cursor-sentinel-replay", "http/complete", "http/explicit-empty", "http/empty", "http/exact-byte-limit", "http/disabled-limit", "http/excess-bytes", "http/invalid-raw-utf8", "http/malformed-encoding", "http/semicolon", "http/unknown", "http/duplicate", "http/invalid-decoded-utf8", "http/empty-field-member", "http/empty-include-member", "http/invalid-filter", "http/empty-sort-name", "http/invalid-page-size", "http/invalid-page-offset", "jsonapi/complete-copy-order", "jsonapi/absent-families", "jsonapi/missing-resource", "jsonapi/missing-filter-decoder", "jsonapi/missing-page-decoder", "jsonapi/filter-nil", "jsonapi/filter-error", "jsonapi/page-error", "jsonapi/callback-panic", "postgres/operator/eq", "postgres/operator/neq", "postgres/operator/lt", "postgres/operator/lte", "postgres/operator/gt", "postgres/operator/gte", "postgres/operator/in", "postgres/operator/not_in", "postgres/operator/between", "postgres/operator/is_null", "postgres/operator/contains", "postgres/operator/starts_with", "postgres/operator/ends_with", "postgres/operator/and", "postgres/operator/or", "postgres/operator/not", "postgres/invalid-identifier/", "postgres/invalid-identifier/.id", "postgres/invalid-identifier/a.b.c.d", "postgres/invalid-identifier/1table.id", "postgres/invalid-identifier/table.bad-name", "postgres/invalid-identifier/table.id;drop", "postgres/empty-capability", "postgres/nil-compiler", "postgres/nil-plan", "postgres/mapping-snapshot-and-constraint-precedence", "postgres/missing-field", "postgres/missing-filter", "postgres/missing-sort", "postgres/missing-constraint", "postgres/real-database", "jsonrpc/absent", "jsonrpc/complete-copy", "jsonrpc/exact-byte-limit", "jsonrpc/explicit-empty", "jsonrpc/invalid/array", "jsonrpc/invalid/disabled-bound", "jsonrpc/invalid/duplicate", "jsonrpc/invalid/excess-bound", "jsonrpc/invalid/excess-depth", "jsonrpc/invalid/invalid-utf8", "jsonrpc/invalid/nested-duplicate", "jsonrpc/invalid/nil", "jsonrpc/invalid/trailing", "jsonrpc/invalid/unknown", "jsonrpc/descriptor-copy", "validation/nil", "validation/generic", "validation/structured-copy", "validation/nested-path", "validation/truncated", "resources/observed-stateless"}
	seenRecords := map[string]struct{}{}
	for index, item := range output.Records {
		expected := contract.Records[index]
		require(item.Name == expectedRecordNames[index] && item.Name == expected.Name && validDigest(expected.ValueSHA256), "output record identity")
		_, duplicate := seenRecords[item.Name]
		require(!duplicate, "duplicate output record")
		seenRecords[item.Name] = struct{}{}
		var value any
		must(json.Unmarshal(item.Value, &value))
		var canonical bytes.Buffer
		encoder := json.NewEncoder(&canonical)
		encoder.SetEscapeHTML(false)
		must(encoder.Encode(value))
		require(digestBytes(canonical.Bytes()) == expected.ValueSHA256, "output exact value "+item.Name)
	}
}

func validateReleaseArtifact(ctx context.Context, root, commit string, item releaseArtifact, expectedBlob string, expectedBytes int, expectedSHA string) {
	require(item.Blob == expectedBlob && item.SHA256 == expectedSHA && (expectedBytes == 0 || item.Bytes == expectedBytes), "release artifact declaration "+item.Path)
	command := exec.CommandContext(ctx, "git", "show", commit+":"+item.Path)
	command.Dir = root
	content, err := command.Output()
	must(err)
	require(digestBytes(content) == item.SHA256 && (expectedBytes == 0 || len(content) == item.Bytes), "release artifact content "+item.Path)
	command = exec.CommandContext(ctx, "git", "rev-parse", commit+":"+item.Path)
	command.Dir = root
	blob, err := command.Output()
	must(err)
	require(strings.TrimSpace(string(blob)) == item.Blob, "release artifact blob "+item.Path)
}

func validateReleaseObjects(ctx context.Context, root string, packet manifest) {
	require(gitOutput(ctx, root, "cat-file", "-t", packet.Release.TagObject) == "tag", "release tag object type")
	require(gitOutput(ctx, root, "rev-parse", "refs/tags/"+release) == packet.Release.TagObject, "release tag reference")
	require(gitOutput(ctx, root, "rev-parse", packet.Release.TagObject+"^{}") == packet.Release.PeeledCommit, "release peeled commit")
	require(gitOutput(ctx, root, "rev-parse", packet.Release.PeeledCommit+"^{tree}") == packet.Release.Tree, "release commit tree")
	allowedSigners := localPath(root, packet.Release.AllowedSigners.Path)
	command := exec.CommandContext(ctx, "git", "-c", "gpg.ssh.allowedSignersFile="+allowedSigners, "verify-tag", packet.Release.TagObject)
	command.Dir = root
	require(command.Run() == nil, "release tag SSH signature")
}

func gitOutput(ctx context.Context, root string, arguments ...string) string {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = root
	output, err := command.Output()
	must(err)
	return strings.TrimSpace(string(output))
}

func validateArtifactSet(root string, artifacts []artifact) {
	expectedPaths := []string{
		".verification/cohesion/api-query-v1-oracle/checkpoints/root.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiqueryhttp.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiqueryjsonapi.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiquerypgx.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiqueryrpc.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiquerytest.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__apiqueryvalidation.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__cursor.json",
		".verification/cohesion/api-query-v1-oracle/checkpoints/root__internal__strictjson.json",
		".verification/cohesion/api-query-v1-oracle/decision-freeze.json",
		".verification/cohesion/api-query-v1-oracle/execution-result.json",
		".verification/cohesion/api-query-v1-oracle/external-go.mod",
		".verification/cohesion/api-query-v1-oracle/external-go.sum",
		".verification/cohesion/api-query-v1-oracle/generate-contract.go",
		".verification/cohesion/api-query-v1-oracle/harness.go",
		".verification/cohesion/api-query-v1-oracle/input.json",
		".verification/cohesion/api-query-v1-oracle/module-graph.txt",
		".verification/cohesion/api-query-v1-oracle/output-contract.json",
		".verification/cohesion/api-query-v1-oracle/output.json",
		".verification/cohesion/api-query-v1-oracle/postgres-container-inspect.json",
		".verification/cohesion/api-query-v1-oracle/postgres-image-inspect.json",
		".verification/cohesion/api-query-v1-oracle/published-module-download.json",
		".verification/cohesion/api-query-v1-oracle/published-module.zip",
		".verification/cohesion/api-query-v1-oracle/release-allowed-signers",
		".verification/cohesion/api-query-v1-oracle/shared-test-input.json",
		".verification/cohesion/api-query-v1-oracle/test-logs/root.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiqueryhttp.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiqueryjsonapi.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiquerypgx.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiqueryrpc.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiquerytest.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__apiqueryvalidation.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__cursor.jsonl",
		".verification/cohesion/api-query-v1-oracle/test-logs/root__internal__strictjson.jsonl",
		".verification/cohesion/api-query-v1-oracle/verify.go",
	}
	require(len(artifacts) == len(expectedPaths), "artifact cardinality")
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(artifacts))
	for _, item := range artifacts {
		validateArtifact(root, item)
		_, duplicate := seen[item.Path]
		require(!duplicate, "duplicate artifact "+item.Path)
		seen[item.Path] = struct{}{}
		paths = append(paths, item.Path)
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	require(equalStrings(paths, sorted), "artifact ordering")
	require(equalStrings(paths, expectedPaths), "artifact path contract")
}

func validateArtifact(root string, item artifact) {
	require(item.Path != "" && validDigest(item.SHA256), "artifact identity")
	require(digestFile(localPath(root, item.Path)) == item.SHA256, "artifact digest "+item.Path)
}

func localPath(root, relative string) string {
	require(!filepath.IsAbs(relative), "absolute artifact path")
	clean := filepath.Clean(relative)
	require(clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)), "artifact path traversal")
	current := root
	parts := strings.Split(clean, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		must(err)
		require(info.Mode()&os.ModeSymlink == 0, "artifact symlink")
		if index < len(parts)-1 {
			require(info.IsDir(), "artifact parent directory")
		} else {
			require(info.Mode().IsRegular(), "artifact regular file")
		}
	}
	return current
}

func decodeClosedFile(filePath string, target any) error {
	data, err := readBoundedFile(filePath, maxEvidenceBytes)
	if err != nil {
		return err
	}
	if err := rejectDuplicateJSONMembers(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func decodeJSONFile(filePath string, target any) error {
	data, err := readBoundedFile(filePath, maxEvidenceBytes)
	if err != nil {
		return err
	}
	if err := rejectDuplicateJSONMembers(data); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func rejectDuplicateJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	if depth > maxJSONDepth {
		return errors.New("JSON exceeds maximum depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	if delimiter == '{' {
		seen := map[string]struct{}{}
		for decoder.More() {
			member, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := member.(string)
			if !ok {
				return errors.New("non-string member")
			}
			if _, duplicate := seen[name]; duplicate {
				return fmt.Errorf("duplicate member %q", name)
			}
			seen[name] = struct{}{}
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else if delimiter == '[' {
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else {
		return errors.New("unexpected delimiter")
	}
	_, err = decoder.Token()
	return err
}

func digestFile(filePath string) string {
	data, err := readBoundedFile(filePath, maxBinaryBytes)
	must(err)
	return digestBytes(data)
}

func readBoundedFile(filePath string, maximum int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, errors.New("file exceeds maximum bytes")
	}
	return data, nil
}

func moduleZipSum(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	if len(reader.File) > maxModuleZipEntries {
		return "", errors.New("module zip exceeds maximum entries")
	}
	files := append([]*zip.File(nil), reader.File...)
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	digest := sha256.New()
	var total uint64
	for _, item := range files {
		if strings.Contains(item.Name, "\n") {
			return "", errors.New("module zip filename contains newline")
		}
		total += item.UncompressedSize64
		if total > maxModuleZipBytes {
			return "", errors.New("module zip exceeds maximum expanded bytes")
		}
		content, err := item.Open()
		if err != nil {
			return "", err
		}
		fileDigest := sha256.New()
		_, copyErr := io.CopyN(fileDigest, content, int64(item.UncompressedSize64))
		closeErr := content.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		fmt.Fprintf(digest, "%x  %s\n", fileDigest.Sum(nil), item.Name)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(digest.Sum(nil)), nil
}
func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}
func equalStrings(left, right []string) bool {
	return strings.Join(left, "\n") == strings.Join(right, "\n")
}
func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
func require(condition bool, label string) {
	if !condition {
		panic(label)
	}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
