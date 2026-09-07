// Command verify reconstructs and executes the final legacy compatibility
// replay from the exact source commit recorded in its closed receipt.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	modulePath       = "github.com/faustbrian/go-api-query"
	candidateVersion = "v1.1.0-replay.0"
	receiptPath      = ".verification/cohesion/api-query-v1-final-replay.json"
	oracleRoot       = ".verification/cohesion/api-query-v1-oracle"
	expectedOutput   = "9da16adc97f2074c3e890a290a8b7028938803fdc702d4f2689cfd8c9cd3816b"
	postgresImage    = "postgres@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15"
	maximumBytes     = 8 << 20
)

type receipt struct {
	Schema     string `json:"schema"`
	CapturedAt string `json:"captured_at"`
	Source     struct {
		Commit string `json:"commit"`
		Tree   string `json:"tree"`
	} `json:"source"`
	Candidate struct {
		Module         string `json:"module"`
		Version        string `json:"version"`
		ProxyZipSHA256 string `json:"proxy_zip_sha256"`
		ProxyModSHA256 string `json:"proxy_mod_sha256"`
		ModuleSum      string `json:"module_sum"`
		GoModSum       string `json:"go_mod_sum"`
	} `json:"candidate"`
	ExternalConsumer struct {
		ReplaceDirectives             int    `json:"replace_directives"`
		WorkspaceFile                 bool   `json:"workspace_file"`
		SelectedVersion               string `json:"selected_version"`
		SelectedFromTaskOwnedModCache bool   `json:"selected_from_task_owned_module_cache"`
		GOPROXY                       string `json:"goproxy"`
		GOSUMDB                       string `json:"gosumdb"`
		GONOSUMDB                     string `json:"gonosumdb"`
		GOPRIVATE                     string `json:"goprivate"`
		GOWORK                        string `json:"gowork"`
	} `json:"external_consumer"`
	Toolchain struct {
		ExecutionIdentity   string `json:"execution_identity"`
		AcceptedExecutables []struct {
			Identity         string `json:"identity"`
			ExecutableSHA256 string `json:"executable_sha256"`
		} `json:"accepted_executables"`
	} `json:"toolchain"`
	Oracle struct {
		HarnessPath          string `json:"harness_path"`
		HarnessSHA256        string `json:"harness_sha256"`
		InputPath            string `json:"input_path"`
		InputSHA256          string `json:"input_sha256"`
		ReleasedOutputPath   string `json:"released_output_path"`
		ReleasedOutputSHA256 string `json:"released_output_sha256"`
		ReplayOutputSHA256   string `json:"replay_output_sha256"`
		ByteComparison       string `json:"byte_comparison"`
		Outcome              string `json:"outcome"`
	} `json:"oracle"`
	Postgres struct {
		RepositoryDigest string `json:"repository_digest"`
		Database         string `json:"database"`
		User             string `json:"user"`
		DSN              string `json:"dsn"`
	} `json:"postgres"`
	Cleanup struct {
		TaskRootRemoved  bool `json:"task_root_removed"`
		ContainerRemoved bool `json:"container_removed"`
		VolumeCreated    bool `json:"volume_created"`
	} `json:"cleanup"`
}

type moduleDownload struct {
	Path       string          `json:"Path"`
	Query      string          `json:"Query"`
	Version    string          `json:"Version"`
	Versions   []string        `json:"Versions"`
	Replace    *moduleDownload `json:"Replace"`
	Time       *time.Time      `json:"Time"`
	Update     *moduleDownload `json:"Update"`
	Main       bool            `json:"Main"`
	Indirect   bool            `json:"Indirect"`
	Dir        string          `json:"Dir"`
	GoMod      string          `json:"GoMod"`
	GoVersion  string          `json:"GoVersion"`
	Retracted  []string        `json:"Retracted"`
	Deprecated string          `json:"Deprecated"`
	Error      *moduleError    `json:"Error"`
	Info       string          `json:"Info"`
	Zip        string          `json:"Zip"`
	Sum        string          `json:"Sum"`
	GoModSum   string          `json:"GoModSum"`
	Reuse      bool            `json:"Reuse"`
	Origin     *struct {
		VCS       string `json:"VCS"`
		URL       string `json:"URL"`
		Subdir    string `json:"Subdir"`
		Hash      string `json:"Hash"`
		TagPrefix string `json:"TagPrefix"`
		TagSum    string `json:"TagSum"`
		Ref       string `json:"Ref"`
		RepoSum   string `json:"RepoSum"`
	} `json:"Origin"`
}

type moduleError struct {
	Err string `json:"Err"`
}

type moduleEdit struct {
	Replace []json.RawMessage `json:"Replace"`
}

type dockerMount struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}

type dockerStorage struct {
	Mounts []dockerMount     `json:"Mounts"`
	Tmpfs  map[string]string `json:"Tmpfs"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	repositoryRoot := strings.TrimSpace(run(ctx, "", nil, "git", "rev-parse", "--show-toplevel"))
	packet := readReceipt(filepath.Join(repositoryRoot, receiptPath))
	validateReceipt(ctx, repositoryRoot, packet)
	replay(ctx, repositoryRoot, packet)

	fmt.Println("api-query final legacy replay is valid")
}

func validateReceipt(ctx context.Context, root string, packet receipt) {
	must(packet.Schema == "api-query-v1-final-replay-v2", "receipt schema")
	_, err := time.Parse(time.RFC3339, packet.CapturedAt)
	check(err)
	must(isHex(packet.Source.Commit, 40) && isHex(packet.Source.Tree, 40), "source object identity")
	must(strings.TrimSpace(run(ctx, root, nil, "git", "rev-parse", packet.Source.Commit+"^{tree}")) == packet.Source.Tree, "source tree")
	run(ctx, root, nil, "git", "merge-base", "--is-ancestor", packet.Source.Commit, "HEAD")
	must(packet.Candidate.Module == modulePath && packet.Candidate.Version == candidateVersion, "candidate identity")
	must(isHex(packet.Candidate.ProxyZipSHA256, 64) && isHex(packet.Candidate.ProxyModSHA256, 64), "candidate digests")
	must(strings.HasPrefix(packet.Candidate.ModuleSum, "h1:") && strings.HasPrefix(packet.Candidate.GoModSum, "h1:"), "candidate sums")
	consumer := packet.ExternalConsumer
	must(consumer.ReplaceDirectives == 0 && !consumer.WorkspaceFile && consumer.SelectedVersion == candidateVersion && consumer.SelectedFromTaskOwnedModCache, "external consumer isolation")
	must(consumer.GOPROXY == "task-owned-file-proxy,https://proxy.golang.org" && consumer.GOSUMDB == "sum.golang.org" && consumer.GONOSUMDB == modulePath && consumer.GOPRIVATE == "" && consumer.GOWORK == "off", "module trust environment")
	must(strings.HasPrefix(packet.Toolchain.ExecutionIdentity, "go version go1.26.6 ") && len(packet.Toolchain.AcceptedExecutables) == 2, "toolchain identity")
	seenToolchains := make(map[string]struct{}, len(packet.Toolchain.AcceptedExecutables))
	for _, toolchain := range packet.Toolchain.AcceptedExecutables {
		must(strings.HasPrefix(toolchain.Identity, "go version go1.26.6 ") && isHex(toolchain.ExecutableSHA256, 64), "accepted toolchain")
		_, duplicate := seenToolchains[toolchain.Identity]
		must(!duplicate, "duplicate toolchain identity")
		seenToolchains[toolchain.Identity] = struct{}{}
	}
	_, executionAccepted := seenToolchains[packet.Toolchain.ExecutionIdentity]
	must(executionAccepted, "execution toolchain acceptance")
	must(packet.Oracle.HarnessPath == oracleRoot+"/harness.go" && packet.Oracle.InputPath == oracleRoot+"/input.json" && packet.Oracle.ReleasedOutputPath == oracleRoot+"/output.json", "oracle paths")
	must(packet.Oracle.ReleasedOutputSHA256 == expectedOutput && packet.Oracle.ReplayOutputSHA256 == expectedOutput && packet.Oracle.ByteComparison == "identical" && packet.Oracle.Outcome == "passed", "oracle outcome")
	must(packet.Postgres.RepositoryDigest == postgresImage && packet.Postgres.Database == "oracle" && packet.Postgres.User == "oracle" && packet.Postgres.DSN == "task-owned-loopback-redacted", "PostgreSQL projection")
	must(packet.Cleanup.TaskRootRemoved && packet.Cleanup.ContainerRemoved && !packet.Cleanup.VolumeCreated, "cleanup outcome")
}

func replay(ctx context.Context, root string, packet receipt) {
	replayRoot, err := os.MkdirTemp("", "go-api-query-final-replay-")
	check(err)
	containerID := ""
	containerCleanupTarget := ""
	defer func() {
		if containerCleanupTarget != "" {
			if err := exec.Command("docker", "rm", "-fv", containerCleanupTarget).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "final replay container cleanup failed: %v\n", err)
			}
		}
		if err := removeTree(replayRoot); err != nil {
			fmt.Fprintf(os.Stderr, "final replay task-root cleanup failed: %v\n", err)
		}
	}()

	proxyVersionRoot := filepath.Join(replayRoot, "proxy", "github.com", "faustbrian", "go-api-query", "@v")
	externalRoot := filepath.Join(replayRoot, "external")
	for _, directory := range []string{proxyVersionRoot, externalRoot, filepath.Join(replayRoot, "cache"), filepath.Join(replayRoot, "mod"), filepath.Join(replayRoot, "tmp")} {
		check(os.MkdirAll(directory, 0o700))
	}

	zipPath := filepath.Join(proxyVersionRoot, candidateVersion+".zip")
	run(ctx, root, nil, "git", "archive", "--format=zip", "--prefix="+modulePath+"@"+candidateVersion+"/", "--output="+zipPath, packet.Source.Commit)
	moduleBytes := gitFile(ctx, root, packet.Source.Commit, "go.mod")
	writeFile(filepath.Join(proxyVersionRoot, candidateVersion+".mod"), moduleBytes)
	commitTime := strings.TrimSpace(run(ctx, root, nil, "git", "show", "-s", "--format=%cI", packet.Source.Commit))
	writeFile(filepath.Join(proxyVersionRoot, candidateVersion+".info"), []byte(fmt.Sprintf("{\"Version\":%q,\"Time\":%q}\n", candidateVersion, commitTime)))
	writeFile(filepath.Join(proxyVersionRoot, "list"), []byte(candidateVersion+"\n"))
	must(digestFile(zipPath) == packet.Candidate.ProxyZipSHA256, "proxy zip digest")
	must(digestBytes(moduleBytes) == packet.Candidate.ProxyModSHA256, "proxy module digest")

	harness := gitFile(ctx, root, packet.Source.Commit, packet.Oracle.HarnessPath)
	input := gitFile(ctx, root, packet.Source.Commit, packet.Oracle.InputPath)
	releasedOutput := gitFile(ctx, root, packet.Source.Commit, packet.Oracle.ReleasedOutputPath)
	must(digestBytes(harness) == packet.Oracle.HarnessSHA256 && digestBytes(input) == packet.Oracle.InputSHA256 && digestBytes(releasedOutput) == expectedOutput, "oracle source bytes")
	writeFile(filepath.Join(externalRoot, "harness.go"), harness)
	writeFile(filepath.Join(externalRoot, "input.json"), input)

	externalMod := gitFile(ctx, root, packet.Source.Commit, oracleRoot+"/external-go.mod")
	releasedRequirement := modulePath + " v1.0.0"
	must(bytes.Count(externalMod, []byte(releasedRequirement)) == 1, "external module requirement")
	externalMod = bytes.Replace(externalMod, []byte(releasedRequirement), []byte(modulePath+" "+candidateVersion), 1)
	writeFile(filepath.Join(externalRoot, "go.mod"), externalMod)
	externalSum := gitFile(ctx, root, packet.Source.Commit, oracleRoot+"/external-go.sum")
	var retainedSum []byte
	for _, line := range bytes.Split(externalSum, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte(modulePath+" v1.0.0")) {
			retainedSum = append(retainedSum, line...)
			retainedSum = append(retainedSum, '\n')
		}
	}
	writeFile(filepath.Join(externalRoot, "go.sum"), retainedSum)
	must(!exists(filepath.Join(externalRoot, "go.work")), "external workspace absence")
	goBinary, err := exec.LookPath("go")
	check(err)
	goBinary, err = filepath.EvalSymlinks(goBinary)
	check(err)
	goVersion := strings.TrimSpace(run(ctx, externalRoot, nil, goBinary, "version"))
	must(strings.HasPrefix(goVersion, "go version go1.26.6 "), "replay Go version")
	acceptedDigest := ""
	for _, toolchain := range packet.Toolchain.AcceptedExecutables {
		if toolchain.Identity == goVersion {
			acceptedDigest = toolchain.ExecutableSHA256
			break
		}
	}
	must(acceptedDigest != "" && digestRegularFile(goBinary) == acceptedDigest, "replay Go executable")
	baseGoEnv := withEnv(os.Environ(),
		"GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=", "GONOPROXY=none",
		"GOCACHE="+filepath.Join(replayRoot, "cache"), "GOMODCACHE="+filepath.Join(replayRoot, "mod"),
		"GOTMPDIR="+filepath.Join(replayRoot, "tmp"), "TMPDIR="+filepath.Join(replayRoot, "tmp"),
		"GOSUMDB=sum.golang.org", "GONOSUMDB="+modulePath, "GOPRIVATE=",
	)
	localProxyEnv := withEnv(baseGoEnv, "GOPROXY=file://"+filepath.Join(replayRoot, "proxy"))
	must(parsedReplaceCount(ctx, externalRoot, localProxyEnv, goBinary) == 0, "external module replace directives")
	moduleCacheEntries, err := os.ReadDir(filepath.Join(replayRoot, "mod"))
	check(err)
	must(len(moduleCacheEntries) == 0, "initially empty module cache")
	downloadOutput := runBytes(ctx, externalRoot, localProxyEnv, goBinary, "mod", "download", "-json", modulePath+"@"+candidateVersion)
	var download moduleDownload
	decodeClosed(downloadOutput, &download)
	must(download.Path == modulePath && download.Version == candidateVersion && download.Sum == packet.Candidate.ModuleSum && download.GoModSum == packet.Candidate.GoModSum && download.Origin == nil, "candidate download receipt")
	must(within(filepath.Join(replayRoot, "mod"), download.Zip) && within(filepath.Join(replayRoot, "mod"), download.GoMod) && digestRegularFile(download.Zip) == packet.Candidate.ProxyZipSHA256 && digestRegularFile(download.GoMod) == packet.Candidate.ProxyModSHA256, "candidate local proxy artifacts")

	envFile := filepath.Join(replayRoot, "postgres.env")
	writeSecretFile(envFile, []byte("POSTGRES_USER=oracle\nPOSTGRES_PASSWORD=oracle_password\nPOSTGRES_DB=oracle\n"))
	containerName := "api-query-final-replay-" + filepath.Base(replayRoot)
	containerID = startOwnedContainer(containerName, &containerCleanupTarget, func() string {
		return strings.TrimSpace(run(ctx, root, nil, "docker", "run", "-d", "--name", containerName, "--env-file", envFile, "--tmpfs", "/var/lib/postgresql:rw,nosuid", "-p", "127.0.0.1::5432", postgresImage))
	})
	must(containerID != "", "PostgreSQL container identity")
	var storage dockerStorage
	decodeClosed(runBytes(ctx, root, nil, "docker", "inspect", "--format", "{\"Mounts\":{{json .Mounts}},\"Tmpfs\":{{json .HostConfig.Tmpfs}}}", containerID), &storage)
	must(isolatedPostgresStorage(storage), "task-owned PostgreSQL tmpfs")
	ready := false
	for attempt := 0; attempt < 60; attempt++ {
		if exec.CommandContext(ctx, "docker", "exec", containerID, "pg_isready", "-U", "oracle", "-d", "oracle").Run() == nil {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	must(ready, "PostgreSQL readiness")
	portOutput := strings.TrimSpace(run(ctx, root, nil, "docker", "port", containerID, "5432/tcp"))
	separator := strings.LastIndexByte(portOutput, ':')
	must(separator >= 0 && separator < len(portOutput)-1, "PostgreSQL port")
	hostPort := portOutput[separator+1:]

	goEnv := withEnv(baseGoEnv,
		"GOPROXY=file://"+filepath.Join(replayRoot, "proxy")+",https://proxy.golang.org",
		"APIQUERY_TEST_DATABASE_URL=postgres://oracle:oracle_password@127.0.0.1:"+hostPort+"/oracle?sslmode=disable",
	)
	run(ctx, externalRoot, goEnv, goBinary, "mod", "tidy")
	must(parsedReplaceCount(ctx, externalRoot, goEnv, goBinary) == 0 && !exists(filepath.Join(externalRoot, "go.work")), "post-resolution isolation")
	selectionOutput := runBytes(ctx, externalRoot, goEnv, goBinary, "list", "-m", "-json", modulePath)
	var selection moduleDownload
	decodeClosed(selectionOutput, &selection)
	must(selection.Path == modulePath && selection.Version == candidateVersion && selection.Replace == nil && within(filepath.Join(replayRoot, "mod"), selection.Dir), "candidate selection")
	graph := run(ctx, externalRoot, goEnv, goBinary, "list", "-m", "all")
	must(strings.Contains(graph, modulePath+" "+candidateVersion), "module graph selection")
	replayOutput := runBytes(ctx, externalRoot, goEnv, goBinary, "run", "./harness.go", "./input.json")
	must(bytes.Equal(releasedOutput, replayOutput) && digestBytes(replayOutput) == packet.Oracle.ReplayOutputSHA256, "byte-identical replay")

	run(ctx, root, nil, "docker", "rm", "-fv", containerID)
	removedID := containerID
	containerID = ""
	containerCleanupTarget = ""
	must(exec.CommandContext(ctx, "docker", "inspect", removedID).Run() != nil, "container cleanup")
	check(removeTree(replayRoot))
	must(!exists(replayRoot), "task root cleanup")
}

func readReceipt(filePath string) receipt {
	data, err := readBounded(filePath)
	check(err)
	var packet receipt
	decodeClosed(data, &packet)
	return packet
}

func decodeClosed(data []byte, target any) {
	check(rejectDuplicateJSONMembers(data))
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	check(decoder.Decode(target))
	var extra any
	must(errors.Is(decoder.Decode(&extra), io.EOF), "single JSON value")
}

func rejectDuplicateJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("JSON exceeds depth bound")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				nameToken, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := nameToken.(string)
				if !ok {
					return errors.New("JSON object member is not a string")
				}
				if _, duplicate := seen[name]; duplicate {
					return fmt.Errorf("duplicate JSON member %q", name)
				}
				seen[name] = struct{}{}
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(0); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func gitFile(ctx context.Context, root, revision, name string) []byte {
	return runBytes(ctx, root, nil, "git", "show", revision+":"+name)
}

func run(ctx context.Context, directory string, environment []string, name string, arguments ...string) string {
	return string(runBytes(ctx, directory, environment, name, arguments...))
}

func runBytes(ctx context.Context, directory string, environment []string, name string, arguments ...string) []byte {
	command := exec.CommandContext(ctx, name, arguments...)
	if directory != "" {
		command.Dir = directory
	}
	if environment != nil {
		command.Env = environment
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedWriter{target: &stdout, remaining: maximumBytes}
	command.Stderr = &limitedWriter{target: &stderr, remaining: maximumBytes}
	if err := command.Run(); err != nil {
		panic(fmt.Sprintf("%s failed: %v: %s", name, err, strings.TrimSpace(stderr.String())))
	}
	return stdout.Bytes()
}

type limitedWriter struct {
	target    *bytes.Buffer
	remaining int
}

func (writer *limitedWriter) Write(data []byte) (int, error) {
	original := len(data)
	if original > writer.remaining {
		return 0, errors.New("command output exceeds bound")
	}
	writer.remaining -= original
	_, err := writer.target.Write(data)
	return original, err
}

func parsedReplaceCount(ctx context.Context, directory string, environment []string, goBinary string) int {
	output := runBytes(ctx, directory, environment, goBinary, "mod", "edit", "-json", "-modfile=go.mod")
	check(rejectDuplicateJSONMembers(output))
	var document moduleEdit
	check(json.Unmarshal(output, &document))
	return len(document.Replace)
}

func readBounded(filePath string) ([]byte, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("file is not regular")
	}
	if info.Size() > maximumBytes {
		return nil, errors.New("file exceeds byte bound")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maximumBytes {
		return nil, errors.New("file exceeds byte bound")
	}
	return data, nil
}

func writeFile(filePath string, data []byte) {
	check(os.WriteFile(filePath, data, 0o600))
}

func writeSecretFile(filePath string, data []byte) {
	check(os.WriteFile(filePath, data, 0o600))
}

func digestFile(filePath string) string {
	data, err := readBounded(filePath)
	check(err)
	return digestBytes(data)
}

func digestRegularFile(filePath string) string {
	info, err := os.Lstat(filePath)
	check(err)
	must(info.Mode().IsRegular(), "executable is not a regular file")
	file, err := os.Open(filePath)
	check(err)
	defer file.Close()
	digest := sha256.New()
	_, err = io.Copy(digest, file)
	check(err)
	return hex.EncodeToString(digest.Sum(nil))
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func within(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func withEnv(base []string, overrides ...string) []string {
	keys := make(map[string]struct{}, len(overrides))
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		keys[key] = struct{}{}
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := keys[key]; !replaced {
			result = append(result, item)
		}
	}
	return append(result, overrides...)
}

func exists(filePath string) bool {
	_, err := os.Lstat(filePath)
	return err == nil
}

func isolatedPostgresStorage(storage dockerStorage) bool {
	options, ok := storage.Tmpfs["/var/lib/postgresql"]
	if !ok || len(storage.Tmpfs) != 1 || len(storage.Mounts) > 1 || !hasOption(options, "rw") || !hasOption(options, "nosuid") {
		return false
	}
	for _, mount := range storage.Mounts {
		if mount.Type != "tmpfs" || mount.Destination != "/var/lib/postgresql" {
			return false
		}
	}
	return true
}

func startOwnedContainer(name string, cleanupTarget *string, start func() string) string {
	*cleanupTarget = name
	containerID := start()
	*cleanupTarget = containerID
	return containerID
}

func hasOption(options, expected string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == expected {
			return true
		}
	}
	return false
}

func removeTree(root string) error {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.RemoveAll(root)
}

func isHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func must(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
