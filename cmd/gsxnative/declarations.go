package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/odvcencio/gosx-native/target"
	"github.com/odvcencio/gosx/nir"
)

type sourceDeclarations struct {
	Routes       []routeDeclaration
	DataLoaders  []endpointDeclaration
	Actions      []endpointDeclaration
	Capabilities []capabilityDeclaration
	Bridges      []bridgeDeclaration
}

func validateProjectDeclarations(cfg *projectConfig, mod *nir.Module) error {
	if cfg == nil {
		return nil
	}
	components := make(map[string]bool, len(mod.Components))
	for _, component := range mod.Components {
		components[component.Name] = true
	}
	seenRoutes := make(map[string]bool, len(cfg.Routes))
	for _, route := range cfg.Routes {
		if !identifierPattern.MatchString(route.Name) {
			return fmt.Errorf("route %q has invalid name", route.Name)
		}
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route %s must use an absolute path", route.Name)
		}
		if route.Component == "" || !components[route.Component] {
			return fmt.Errorf("route %s references unknown component %q", route.Name, route.Component)
		}
		if seenRoutes[route.Name] {
			return fmt.Errorf("duplicate route declaration %q", route.Name)
		}
		seenRoutes[route.Name] = true
		if _, err := parseSourceAuth(route.Auth); err != nil {
			return fmt.Errorf("route %s has invalid auth policy: %w", route.Name, err)
		}
		for _, param := range route.Params {
			if !identifierPattern.MatchString(param.Name) {
				return fmt.Errorf("route %s has invalid param %q", route.Name, param.Name)
			}
			if !supportedDeclarationType(param.Type) {
				return fmt.Errorf("route %s param %s has unsupported type %q", route.Name, param.Name, param.Type)
			}
		}
	}
	if err := validateEndpoints("data loader", cfg.DataLoaders); err != nil {
		return err
	}
	if err := validateEndpoints("action", cfg.Actions); err != nil {
		return err
	}
	if err := validateCapabilities(cfg.Capabilities); err != nil {
		return err
	}
	if err := validateBridges(cfg.Bridges); err != nil {
		return err
	}
	return validateActionInvalidates(cfg)
}

func effectiveProjectConfigForSource(cfg *projectConfig, sourcePath string) (*projectConfig, error) {
	sourceDecls, err := parseSourceDeclarationsFile(sourcePath)
	if err != nil {
		return nil, err
	}
	if cfg == nil && !sourceDecls.hasAny() {
		return nil, nil
	}
	effective := cloneProjectConfig(cfg)
	if sourceDecls.hasRoutes() {
		effective.Routes = sourceDecls.Routes
	}
	if sourceDecls.hasDataLoaders() {
		effective.DataLoaders = sourceDecls.DataLoaders
	}
	if sourceDecls.hasActions() {
		effective.Actions = sourceDecls.Actions
	}
	if sourceDecls.hasCapabilities() {
		effective.Capabilities = sourceDecls.Capabilities
	}
	if sourceDecls.hasBridges() {
		effective.Bridges = sourceDecls.Bridges
	}
	return effective, nil
}

func cloneProjectConfig(cfg *projectConfig) *projectConfig {
	if cfg == nil {
		return &projectConfig{}
	}
	clone := *cfg
	clone.Routes = append([]routeDeclaration(nil), cfg.Routes...)
	clone.DataLoaders = append([]endpointDeclaration(nil), cfg.DataLoaders...)
	clone.Actions = append([]endpointDeclaration(nil), cfg.Actions...)
	clone.Capabilities = append([]capabilityDeclaration(nil), cfg.Capabilities...)
	clone.Bridges = append([]bridgeDeclaration(nil), cfg.Bridges...)
	return &clone
}

func (d sourceDeclarations) hasAny() bool {
	return d.hasRoutes() || d.hasDataLoaders() || d.hasActions() || d.hasCapabilities() || d.hasBridges()
}

func (d sourceDeclarations) hasRoutes() bool {
	return len(d.Routes) > 0
}

func (d sourceDeclarations) hasDataLoaders() bool {
	return len(d.DataLoaders) > 0
}

func (d sourceDeclarations) hasActions() bool {
	return len(d.Actions) > 0
}

func (d sourceDeclarations) hasCapabilities() bool {
	return len(d.Capabilities) > 0
}

func (d sourceDeclarations) hasBridges() bool {
	return len(d.Bridges) > 0
}

func parseSourceDeclarationsFile(path string) (sourceDeclarations, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sourceDeclarations{}, fmt.Errorf("read source declarations %s: %w", path, err)
	}
	decls, err := parseSourceDeclarations(data)
	if err != nil {
		return sourceDeclarations{}, fmt.Errorf("%s: %w", path, err)
	}
	return decls, nil
}

func parseSourceDeclarations(src []byte) (sourceDeclarations, error) {
	var decls sourceDeclarations
	scanner := bufio.NewScanner(bytes.NewReader(src))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "//gosx:") {
			continue
		}
		kind, fields, err := parseSourceDeclaration(line)
		if err != nil {
			return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		switch kind {
		case "route":
			route, err := sourceRouteDeclaration(fields)
			if err != nil {
				return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			decls.Routes = append(decls.Routes, route)
		case "data":
			endpoint, err := sourceEndpointDeclaration("data", fields)
			if err != nil {
				return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			decls.DataLoaders = append(decls.DataLoaders, endpoint)
		case "action":
			endpoint, err := sourceEndpointDeclaration("action", fields)
			if err != nil {
				return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			decls.Actions = append(decls.Actions, endpoint)
		case "capability":
			capability, err := sourceCapabilityDeclaration(fields)
			if err != nil {
				return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			decls.Capabilities = append(decls.Capabilities, capability)
		case "bridge":
			bridge, err := sourceBridgeDeclaration(fields)
			if err != nil {
				return sourceDeclarations{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			decls.Bridges = append(decls.Bridges, bridge)
		default:
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return sourceDeclarations{}, err
	}
	return decls, nil
}

func parseSourceDeclaration(line string) (string, map[string][]string, error) {
	body := strings.TrimSpace(strings.TrimPrefix(line, "//gosx:"))
	pieces := strings.Fields(body)
	if len(pieces) == 0 {
		return "", nil, fmt.Errorf("empty gosx declaration")
	}
	kind := strings.ToLower(pieces[0])
	if kind != "route" && kind != "data" && kind != "action" && kind != "capability" && kind != "bridge" {
		return kind, nil, nil
	}
	fields := make(map[string][]string, len(pieces)-1)
	for _, piece := range pieces[1:] {
		name, value, ok := strings.Cut(piece, "=")
		if !ok || strings.TrimSpace(name) == "" {
			return "", nil, fmt.Errorf("declaration field %q must use key=value", piece)
		}
		name = strings.ToLower(strings.TrimSpace(name))
		fields[name] = append(fields[name], strings.TrimSpace(value))
	}
	return kind, fields, nil
}

func sourceRouteDeclaration(fields map[string][]string) (routeDeclaration, error) {
	if err := requireDeclarationFields("route", fields, "name", "path", "component"); err != nil {
		return routeDeclaration{}, err
	}
	if err := rejectUnknownDeclarationFields("route", fields, "name", "path", "component", "params", "param", "auth"); err != nil {
		return routeDeclaration{}, err
	}
	params, err := parseSourceParams(append(fields["params"], fields["param"]...))
	if err != nil {
		return routeDeclaration{}, err
	}
	auth, err := parseSourceAuth(firstField(fields, "auth"))
	if err != nil {
		return routeDeclaration{}, err
	}
	return routeDeclaration{
		Name:      firstField(fields, "name"),
		Path:      firstField(fields, "path"),
		Component: firstField(fields, "component"),
		Params:    params,
		Auth:      auth,
	}, nil
}

func sourceEndpointDeclaration(kind string, fields map[string][]string) (endpointDeclaration, error) {
	if err := requireDeclarationFields(kind, fields, "name", "method", "path"); err != nil {
		return endpointDeclaration{}, err
	}
	if err := rejectUnknownDeclarationFields(kind, fields,
		"name", "method", "path",
		"params", "param",
		"input", "request", "body",
		"output", "response", "returns",
		"ttl", "cache_ttl", "cache_ttl_seconds",
		"invalidates", "invalidate",
		"optimistic", "auth",
		"retry", "retries", "retry_attempts",
	); err != nil {
		return endpointDeclaration{}, err
	}
	params, err := parseSourceParams(append(fields["params"], fields["param"]...))
	if err != nil {
		return endpointDeclaration{}, err
	}
	input, err := parseSourceParams(append(append(fields["input"], fields["request"]...), fields["body"]...))
	if err != nil {
		return endpointDeclaration{}, err
	}
	output, err := parseSourceParams(append(append(fields["output"], fields["response"]...), fields["returns"]...))
	if err != nil {
		return endpointDeclaration{}, err
	}
	cacheTTL, err := parseSourceSeconds(firstNonEmptyField(fields, "cache_ttl_seconds", "cache_ttl", "ttl"))
	if err != nil {
		return endpointDeclaration{}, err
	}
	retryAttempts, err := parseSourcePositiveInt(firstNonEmptyField(fields, "retry_attempts", "retry", "retries"))
	if err != nil {
		return endpointDeclaration{}, err
	}
	auth, err := parseSourceAuth(firstField(fields, "auth"))
	if err != nil {
		return endpointDeclaration{}, err
	}
	return endpointDeclaration{
		Name:            firstField(fields, "name"),
		Method:          firstField(fields, "method"),
		Path:            firstField(fields, "path"),
		Params:          params,
		Input:           input,
		Output:          output,
		CacheTTLSeconds: cacheTTL,
		Invalidates:     parseSourceList(append(fields["invalidates"], fields["invalidate"]...)),
		Optimistic:      firstField(fields, "optimistic"),
		Auth:            auth,
		RetryAttempts:   retryAttempts,
	}, nil
}

func sourceCapabilityDeclaration(fields map[string][]string) (capabilityDeclaration, error) {
	if err := requireDeclarationFields("capability", fields, "name"); err != nil {
		return capabilityDeclaration{}, err
	}
	if err := rejectUnknownDeclarationFields("capability", fields, "name", "target", "targets", "required"); err != nil {
		return capabilityDeclaration{}, err
	}
	required, err := parseSourceBool(firstField(fields, "required"))
	if err != nil {
		return capabilityDeclaration{}, err
	}
	return capabilityDeclaration{
		Name:     firstField(fields, "name"),
		Targets:  parseCapabilityTargets(append(fields["targets"], fields["target"]...)),
		Required: required,
	}, nil
}

func sourceBridgeDeclaration(fields map[string][]string) (bridgeDeclaration, error) {
	if err := requireDeclarationFields("bridge", fields, "service", "method", "path"); err != nil {
		return bridgeDeclaration{}, err
	}
	if err := rejectUnknownDeclarationFields("bridge", fields,
		"service", "method", "path",
		"input", "request", "body",
		"output", "response", "returns",
		"auth",
		"retry", "retries", "retry_attempts",
	); err != nil {
		return bridgeDeclaration{}, err
	}
	input, err := parseSourceParams(append(append(fields["input"], fields["request"]...), fields["body"]...))
	if err != nil {
		return bridgeDeclaration{}, err
	}
	output, err := parseSourceParams(append(append(fields["output"], fields["response"]...), fields["returns"]...))
	if err != nil {
		return bridgeDeclaration{}, err
	}
	retryAttempts, err := parseSourcePositiveInt(firstNonEmptyField(fields, "retry_attempts", "retry", "retries"))
	if err != nil {
		return bridgeDeclaration{}, err
	}
	auth, err := parseSourceAuth(firstField(fields, "auth"))
	if err != nil {
		return bridgeDeclaration{}, err
	}
	return bridgeDeclaration{
		Service:       firstField(fields, "service"),
		Method:        firstField(fields, "method"),
		Path:          firstField(fields, "path"),
		Input:         input,
		Output:        output,
		Auth:          auth,
		RetryAttempts: retryAttempts,
	}, nil
}

func requireDeclarationFields(kind string, fields map[string][]string, required ...string) error {
	for _, name := range required {
		if firstField(fields, name) == "" {
			return fmt.Errorf("%s declaration missing %s", kind, name)
		}
	}
	return nil
}

func rejectUnknownDeclarationFields(kind string, fields map[string][]string, allowed ...string) error {
	known := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		known[name] = true
	}
	for name := range fields {
		if !known[name] {
			return fmt.Errorf("%s declaration has unknown field %s", kind, name)
		}
	}
	return nil
}

func firstField(fields map[string][]string, name string) string {
	values := fields[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstNonEmptyField(fields map[string][]string, names ...string) string {
	for _, name := range names {
		if value := firstField(fields, name); value != "" {
			return value
		}
	}
	return ""
}

func parseSourceParams(values []string) ([]paramDeclaration, error) {
	var params []paramDeclaration
	for _, value := range values {
		for _, piece := range strings.Split(value, ",") {
			piece = strings.TrimSpace(piece)
			if piece == "" {
				continue
			}
			name, typ, ok := strings.Cut(piece, ":")
			if !ok {
				name = piece
				typ = "string"
			}
			name = strings.TrimSpace(name)
			typ = strings.TrimSpace(typ)
			if typ == "" {
				typ = "string"
			}
			params = append(params, paramDeclaration{Name: name, Type: typ})
		}
	}
	return params, nil
}

func parseSourceList(values []string) []string {
	var out []string
	for _, value := range values {
		for _, piece := range strings.Split(value, ",") {
			piece = strings.TrimSpace(piece)
			if piece != "" {
				out = append(out, piece)
			}
		}
	}
	return out
}

func parseCapabilityTargets(values []string) []string {
	targets := parseSourceList(values)
	if len(targets) == 0 {
		return nil
	}
	for i, value := range targets {
		targets[i] = strings.ToLower(strings.TrimSpace(value))
	}
	return targets
}

func parseSourceSeconds(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("cache TTL must be non-negative")
		}
		return seconds, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("cache TTL %q must be seconds or a Go duration", value)
	}
	if duration < 0 {
		return 0, fmt.Errorf("cache TTL must be non-negative")
	}
	return int(duration / time.Second), nil
}

func parseSourcePositiveInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("retry attempts %q must be a non-negative integer", value)
	}
	return parsed, nil
}

func parseSourceBool(value string) (bool, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "false", "no", "0":
		return false, nil
	case "true", "yes", "1":
		return true, nil
	default:
		return false, fmt.Errorf("boolean value %q must be true or false", value)
	}
}

func parseSourceAuth(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "none", "optional", "required":
		return value, nil
	default:
		return "", fmt.Errorf("auth policy %q must be none, optional, or required", value)
	}
}

func validateEndpoints(kind string, endpoints []endpointDeclaration) error {
	seen := make(map[string]bool, len(endpoints))
	for _, endpoint := range endpoints {
		if !identifierPattern.MatchString(endpoint.Name) {
			return fmt.Errorf("%s %q has invalid name", kind, endpoint.Name)
		}
		if endpoint.Path == "" || !strings.HasPrefix(endpoint.Path, "/") {
			return fmt.Errorf("%s %s must use an absolute path", kind, endpoint.Name)
		}
		if method := strings.ToUpper(endpoint.Method); method != "" && !supportedHTTPMethod(method) {
			return fmt.Errorf("%s %s has unsupported method %q", kind, endpoint.Name, endpoint.Method)
		}
		if seen[endpoint.Name] {
			return fmt.Errorf("duplicate %s declaration %q", kind, endpoint.Name)
		}
		seen[endpoint.Name] = true
		if err := validateEndpointParams(kind, endpoint.Name, "param", endpoint.Params); err != nil {
			return err
		}
		if err := validateEndpointParams(kind, endpoint.Name, "input field", endpoint.Input); err != nil {
			return err
		}
		if err := validateEndpointParams(kind, endpoint.Name, "output field", endpoint.Output); err != nil {
			return err
		}
		if err := validateEndpointMethodParams(kind, endpoint); err != nil {
			return err
		}
		for _, invalidated := range endpoint.Invalidates {
			if !identifierPattern.MatchString(invalidated) {
				return fmt.Errorf("%s %s invalidates unsupported data loader name %q", kind, endpoint.Name, invalidated)
			}
		}
		if endpoint.Optimistic != "" && !identifierPattern.MatchString(endpoint.Optimistic) {
			return fmt.Errorf("%s %s has invalid optimistic metadata %q", kind, endpoint.Name, endpoint.Optimistic)
		}
		if _, err := parseSourceAuth(endpoint.Auth); err != nil {
			return fmt.Errorf("%s %s has invalid auth policy: %w", kind, endpoint.Name, err)
		}
		if endpoint.CacheTTLSeconds < 0 {
			return fmt.Errorf("%s %s cache TTL must be non-negative", kind, endpoint.Name)
		}
		if endpoint.RetryAttempts < 0 {
			return fmt.Errorf("%s %s retry attempts must be non-negative", kind, endpoint.Name)
		}
	}
	return nil
}

func validateCapabilities(capabilities []capabilityDeclaration) error {
	seen := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		if !identifierPattern.MatchString(capability.Name) {
			return fmt.Errorf("capability %q has invalid name", capability.Name)
		}
		if seen[capability.Name] {
			return fmt.Errorf("duplicate capability declaration %q", capability.Name)
		}
		seen[capability.Name] = true
		targets := capabilityTargets(capability)
		targetSeen := make(map[string]bool, len(targets))
		for _, tgt := range targets {
			if tgt != "ios" && tgt != "android" {
				return fmt.Errorf("capability %s has unsupported target %q", capability.Name, tgt)
			}
			if targetSeen[tgt] {
				return fmt.Errorf("capability %s repeats target %q", capability.Name, tgt)
			}
			targetSeen[tgt] = true
		}
	}
	return nil
}

func validateBridges(bridges []bridgeDeclaration) error {
	seen := make(map[string]bool, len(bridges))
	for _, bridge := range bridges {
		if !identifierPattern.MatchString(bridge.Service) {
			return fmt.Errorf("bridge service %q has invalid name", bridge.Service)
		}
		if !identifierPattern.MatchString(bridge.Method) {
			return fmt.Errorf("bridge %s has invalid method %q", bridge.Service, bridge.Method)
		}
		if bridge.Path == "" || !strings.HasPrefix(bridge.Path, "/") {
			return fmt.Errorf("bridge %s.%s must use an absolute path", bridge.Service, bridge.Method)
		}
		name := bridgeName(bridge)
		if seen[name] {
			return fmt.Errorf("duplicate bridge declaration %q", name)
		}
		seen[name] = true
		if err := validateEndpointParams("bridge", name, "input field", bridge.Input); err != nil {
			return err
		}
		if err := validateEndpointParams("bridge", name, "output field", bridge.Output); err != nil {
			return err
		}
		if err := validateBridgeMethodParams(bridge); err != nil {
			return err
		}
		if _, err := parseSourceAuth(bridge.Auth); err != nil {
			return fmt.Errorf("bridge %s has invalid auth policy: %w", name, err)
		}
		if bridge.RetryAttempts < 0 {
			return fmt.Errorf("bridge %s retry attempts must be non-negative", name)
		}
	}
	return nil
}

func validateEndpointParams(kind, endpointName, label string, params []paramDeclaration) error {
	seen := make(map[string]bool, len(params))
	for _, param := range params {
		if !identifierPattern.MatchString(param.Name) {
			return fmt.Errorf("%s %s has invalid %s %q", kind, endpointName, label, param.Name)
		}
		if !supportedDeclarationType(param.Type) {
			return fmt.Errorf("%s %s %s %s has unsupported type %q", kind, endpointName, label, param.Name, param.Type)
		}
		if seen[param.Name] {
			return fmt.Errorf("%s %s has duplicate %s %q", kind, endpointName, label, param.Name)
		}
		seen[param.Name] = true
	}
	return nil
}

func validateBridgeMethodParams(bridge bridgeDeclaration) error {
	seen := make(map[string]bool, len(bridge.Input))
	for _, input := range bridge.Input {
		if seen[input.Name] {
			return fmt.Errorf("bridge %s uses duplicate generated method parameter %q", bridgeName(bridge), input.Name)
		}
		seen[input.Name] = true
	}
	return nil
}

func validateEndpointMethodParams(kind string, endpoint endpointDeclaration) error {
	seen := make(map[string]string, len(endpoint.Params)+len(endpoint.Input))
	for _, param := range endpoint.Params {
		seen[param.Name] = "param"
	}
	for _, input := range endpoint.Input {
		if previous := seen[input.Name]; previous != "" {
			return fmt.Errorf("%s %s uses %s %q and input field %q as generated method parameters", kind, endpoint.Name, previous, input.Name, input.Name)
		}
		seen[input.Name] = "input field"
	}
	return nil
}

func validateActionInvalidates(cfg *projectConfig) error {
	loaders := make(map[string]bool, len(cfg.DataLoaders))
	for _, loader := range cfg.DataLoaders {
		loaders[loader.Name] = true
	}
	for _, action := range cfg.Actions {
		seen := make(map[string]bool, len(action.Invalidates))
		for _, invalidated := range action.Invalidates {
			if !loaders[invalidated] {
				return fmt.Errorf("action %s invalidates unknown data loader %q", action.Name, invalidated)
			}
			if seen[invalidated] {
				return fmt.Errorf("action %s invalidates data loader %q more than once", action.Name, invalidated)
			}
			seen[invalidated] = true
		}
	}
	return nil
}

func supportedDeclarationType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "string", "int", "double", "float", "bool", "boolean":
		return true
	default:
		return false
	}
}

func supportedHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func emitDeclarationSupport(tgt target.Target, cfg *projectConfig) ([]byte, error) {
	switch tgt {
	case target.IOS:
		return emitSwiftDeclarations(cfg), nil
	case target.Android:
		return emitKotlinDeclarations(cfg), nil
	default:
		return nil, fmt.Errorf("unknown target: %s", tgt)
	}
}

func defaultSupportOutput(tgt target.Target, componentOutput string) string {
	switch tgt {
	case target.IOS:
		return filepath.Join(filepath.Dir(componentOutput), "GSXDeclarations.g.swift")
	case target.Android:
		return filepath.Join(filepath.Dir(componentOutput), "GSXDeclarations.kt")
	default:
		return ""
	}
}

func emitSwiftDeclarations(cfg *projectConfig) []byte {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "// Code generated by gsxnative. DO NOT EDIT.")
	fmt.Fprintln(&buf, "import Foundation")
	fmt.Fprintln(&buf, "import GSXNativeKit")
	fmt.Fprintln(&buf)
	emitSwiftGeneratedSpecs(&buf)
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "public struct GSXGeneratedRouteSpec: Equatable {")
	fmt.Fprintln(&buf, "    public let name: String")
	fmt.Fprintln(&buf, "    public let path: String")
	fmt.Fprintln(&buf, "    public let component: String")
	fmt.Fprintln(&buf, "    public let params: [String]")
	fmt.Fprintln(&buf, "    public let auth: GSXAuthRequirement")
	fmt.Fprintln(&buf, "}")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "public enum GSXRoutes {")
	fmt.Fprintln(&buf, "    public static let specs: [GSXGeneratedRouteSpec] = [")
	for _, route := range cfg.Routes {
		fmt.Fprintf(&buf, "        GSXGeneratedRouteSpec(name: %s, path: %s, component: %s, params: %s, auth: %s),\n",
			strconv.Quote(route.Name), strconv.Quote(route.Path), strconv.Quote(route.Component), swiftStringArray(paramNames(route.Params)), swiftAuthRequirement(route.Auth))
	}
	fmt.Fprintln(&buf, "    ]")
	if len(cfg.Routes) > 0 {
		fmt.Fprintln(&buf)
	}
	for _, route := range cfg.Routes {
		emitSwiftRoute(&buf, route)
	}
	fmt.Fprintln(&buf, "}")
	fmt.Fprintln(&buf)
	emitSwiftEndpointSpecs(&buf, "GSXDataLoaders", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitSwiftEndpointSpecs(&buf, "GSXActions", cfg.Actions, "POST")
	fmt.Fprintln(&buf)
	emitSwiftCapabilitySpecs(&buf, cfg.Capabilities)
	fmt.Fprintln(&buf)
	emitSwiftBridgeSpecs(&buf, cfg.Bridges)
	fmt.Fprintln(&buf)
	emitSwiftEndpointClient(&buf, "GSXGeneratedDataClient", "load", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitSwiftEndpointClient(&buf, "GSXGeneratedActionClient", "submit", cfg.Actions, "POST")
	fmt.Fprintln(&buf)
	emitSwiftBridgeClient(&buf, cfg.Bridges)
	return buf.Bytes()
}

func emitSwiftGeneratedSpecs(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "public struct GSXGeneratedParamSpec: Equatable {")
	fmt.Fprintln(buf, "    public let name: String")
	fmt.Fprintln(buf, "    public let type: String")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "public struct GSXGeneratedEndpointSpec: Equatable {")
	fmt.Fprintln(buf, "    public let name: String")
	fmt.Fprintln(buf, "    public let method: String")
	fmt.Fprintln(buf, "    public let path: String")
	fmt.Fprintln(buf, "    public let params: [GSXGeneratedParamSpec]")
	fmt.Fprintln(buf, "    public let input: [GSXGeneratedParamSpec]")
	fmt.Fprintln(buf, "    public let output: [GSXGeneratedParamSpec]")
	fmt.Fprintln(buf, "    public let cacheTTLSeconds: Int?")
	fmt.Fprintln(buf, "    public let invalidates: [String]")
	fmt.Fprintln(buf, "    public let optimistic: String?")
	fmt.Fprintln(buf, "    public let auth: GSXAuthRequirement")
	fmt.Fprintln(buf, "    public let retryAttempts: Int?")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "public struct GSXGeneratedCapabilitySpec: Equatable {")
	fmt.Fprintln(buf, "    public let name: String")
	fmt.Fprintln(buf, "    public let targets: [String]")
	fmt.Fprintln(buf, "    public let required: Bool")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "public struct GSXGeneratedBridgeSpec: Equatable {")
	fmt.Fprintln(buf, "    public let service: String")
	fmt.Fprintln(buf, "    public let method: String")
	fmt.Fprintln(buf, "    public let path: String")
	fmt.Fprintln(buf, "    public let input: [GSXGeneratedParamSpec]")
	fmt.Fprintln(buf, "    public let output: [GSXGeneratedParamSpec]")
	fmt.Fprintln(buf, "    public let auth: GSXAuthRequirement")
	fmt.Fprintln(buf, "    public let retryAttempts: Int?")
	fmt.Fprintln(buf, "}")
}

func emitSwiftRoute(buf *bytes.Buffer, route routeDeclaration) {
	name := swiftIdentifier(route.Name)
	if len(route.Params) == 0 {
		fmt.Fprintf(buf, "    public static let %s = GSXRoute(%s, auth: %s)\n", name, strconv.Quote(route.Name), swiftAuthRequirement(route.Auth))
		return
	}
	fmt.Fprintf(buf, "    public static func %s(%s) -> GSXRoute {\n", name, swiftParamList(route.Params))
	fmt.Fprintln(buf, "        GSXRoute(")
	fmt.Fprintf(buf, "            %s,\n", strconv.Quote(route.Name))
	fmt.Fprintln(buf, "            params: [")
	for _, param := range route.Params {
		fmt.Fprintf(buf, "                %s: %s,\n", strconv.Quote(param.Name), swiftStringValue(param.Name, param.Type))
	}
	fmt.Fprintln(buf, "            ],")
	fmt.Fprintf(buf, "            auth: %s\n", swiftAuthRequirement(route.Auth))
	fmt.Fprintln(buf, "        )")
	fmt.Fprintln(buf, "    }")
}

func emitSwiftEndpointSpecs(buf *bytes.Buffer, enumName string, endpoints []endpointDeclaration, defaultMethod string) {
	fmt.Fprintf(buf, "public enum %s {\n", enumName)
	fmt.Fprintln(buf, "    public static let specs: [GSXGeneratedEndpointSpec] = [")
	for _, endpoint := range endpoints {
		fmt.Fprintf(buf, "        GSXGeneratedEndpointSpec(name: %s, method: %s, path: %s, params: %s, input: %s, output: %s, cacheTTLSeconds: %s, invalidates: %s, optimistic: %s, auth: %s, retryAttempts: %s),\n",
			strconv.Quote(endpoint.Name),
			strconv.Quote(endpointMethod(endpoint, defaultMethod)),
			strconv.Quote(endpoint.Path),
			swiftParamSpecArray(endpoint.Params),
			swiftParamSpecArray(endpoint.Input),
			swiftParamSpecArray(endpoint.Output),
			swiftOptionalInt(endpoint.CacheTTLSeconds),
			swiftStringArray(endpoint.Invalidates),
			swiftOptionalString(endpoint.Optimistic),
			swiftAuthRequirement(endpoint.Auth),
			swiftOptionalInt(endpoint.RetryAttempts),
		)
	}
	fmt.Fprintln(buf, "    ]")
	fmt.Fprintln(buf, "}")
}

func emitSwiftCapabilitySpecs(buf *bytes.Buffer, capabilities []capabilityDeclaration) {
	fmt.Fprintln(buf, "public enum GSXCapabilities {")
	fmt.Fprintln(buf, "    public static let specs: [GSXGeneratedCapabilitySpec] = [")
	for _, capability := range capabilities {
		fmt.Fprintf(buf, "        GSXGeneratedCapabilitySpec(name: %s, targets: %s, required: %s),\n",
			strconv.Quote(capability.Name),
			swiftStringArray(capabilityTargets(capability)),
			swiftBool(capability.Required),
		)
	}
	fmt.Fprintln(buf, "    ]")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public static let runtimeSpecs: [GSXCapabilitySpec] = specs.map { spec in")
	fmt.Fprintln(buf, "        GSXCapabilitySpec(name: spec.name, targets: spec.targets, required: spec.required)")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public static func negotiate(")
	fmt.Fprintln(buf, "        with negotiator: GSXCapabilityNegotiator,")
	fmt.Fprintln(buf, "        target: String = \"ios\",")
	fmt.Fprintln(buf, "        path: String = \"/api/capabilities\"")
	fmt.Fprintln(buf, "    ) async throws -> GSXCapabilityReport {")
	fmt.Fprintln(buf, "        try await negotiator.negotiate(required: runtimeSpecs, target: target, path: path)")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf, "}")
}

func emitSwiftBridgeSpecs(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	fmt.Fprintln(buf, "public enum GSXBridges {")
	fmt.Fprintln(buf, "    public static let specs: [GSXGeneratedBridgeSpec] = [")
	for _, bridge := range bridges {
		fmt.Fprintf(buf, "        GSXGeneratedBridgeSpec(service: %s, method: %s, path: %s, input: %s, output: %s, auth: %s, retryAttempts: %s),\n",
			strconv.Quote(bridge.Service),
			strconv.Quote(bridge.Method),
			strconv.Quote(bridge.Path),
			swiftParamSpecArray(bridge.Input),
			swiftParamSpecArray(bridge.Output),
			swiftAuthRequirement(bridge.Auth),
			swiftOptionalInt(bridge.RetryAttempts),
		)
	}
	fmt.Fprintln(buf, "    ]")
	fmt.Fprintln(buf, "}")
}

func emitSwiftEndpointClient(buf *bytes.Buffer, className, operation string, endpoints []endpointDeclaration, defaultMethod string) {
	emitSwiftEndpointModels(buf, className, endpoints)
	if len(endpoints) > 0 {
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "public final class %s {\n", className)
	fmt.Fprintln(buf, "    private let client: GSXDataClient")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public init(client: GSXDataClient) {")
	fmt.Fprintln(buf, "        self.client = client")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(transport: any GSXTransport) {")
	fmt.Fprintln(buf, "        self.init(client: GSXDataClient(transport: transport))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:]) {")
	fmt.Fprintln(buf, "        self.init(client: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {")
	fmt.Fprintln(buf, "        self.init(client: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {")
	fmt.Fprintln(buf, "        self.init(client: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {")
	fmt.Fprintln(buf, "        self.init(client: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))")
	fmt.Fprintln(buf, "    }")
	for _, endpoint := range endpoints {
		fmt.Fprintln(buf)
		resultType := "GSXResponse"
		outputModel := swiftEndpointModelName(className, endpoint, "Response")
		if len(endpoint.Output) > 0 {
			resultType = outputModel
		}
		fmt.Fprintf(buf, "    public func %s(%s) async throws -> %s {\n", swiftIdentifier(endpoint.Name), swiftEndpointParamList(endpoint), resultType)
		emitSwiftEndpointRequest(buf, className, endpoint, defaultMethod)
		fmt.Fprintf(buf, "        let response = try await client.%s(request, policy: %s)\n", operation, swiftRequestPolicy(endpoint))
		if len(endpoint.Output) > 0 {
			fmt.Fprintf(buf, "        return try response.decodedJSON(%s.self)\n", outputModel)
		} else {
			fmt.Fprintln(buf, "        return response")
		}
		fmt.Fprintln(buf, "    }")
	}
	fmt.Fprintln(buf, "}")
}

func emitSwiftEndpointModels(buf *bytes.Buffer, className string, endpoints []endpointDeclaration) {
	for _, endpoint := range endpoints {
		if len(endpoint.Input) > 0 {
			emitSwiftEndpointModel(buf, swiftEndpointModelName(className, endpoint, "Input"), endpoint.Input)
			fmt.Fprintln(buf)
		}
		if len(endpoint.Output) > 0 {
			emitSwiftEndpointModel(buf, swiftEndpointModelName(className, endpoint, "Response"), endpoint.Output)
			fmt.Fprintln(buf)
		}
	}
}

func emitSwiftEndpointModel(buf *bytes.Buffer, name string, fields []paramDeclaration) {
	fmt.Fprintf(buf, "public struct %s: Codable, Equatable {\n", name)
	for _, field := range fields {
		fmt.Fprintf(buf, "    public let %s: %s\n", swiftIdentifier(field.Name), swiftTypeForDecl(field.Type))
	}
	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "    public init(%s) {\n", swiftParamList(fields))
	for _, field := range fields {
		identifier := swiftIdentifier(field.Name)
		fmt.Fprintf(buf, "        self.%s = %s\n", identifier, identifier)
	}
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf, "}")
}

func emitSwiftEndpointRequest(buf *bytes.Buffer, className string, endpoint endpointDeclaration, defaultMethod string) {
	pathExpr := strconv.Quote(endpoint.Path)
	if len(endpoint.Params) > 0 {
		fmt.Fprintf(buf, "        let path = GSXRequest.resolvedPath(%s, params: %s)\n", strconv.Quote(endpoint.Path), swiftParamMap(endpoint.Params))
		pathExpr = "path"
	}
	if len(endpoint.Input) > 0 {
		fmt.Fprintf(buf, "        let input = %s(%s)\n", swiftEndpointModelName(className, endpoint, "Input"), swiftModelInitArgs(endpoint.Input))
		fmt.Fprintf(buf, "        let request = try GSXRequest.json(method: %s, path: %s, body: input)\n", strconv.Quote(endpointMethod(endpoint, defaultMethod)), pathExpr)
		return
	}
	fmt.Fprintf(buf, "        let request = GSXRequest(method: %s, path: %s)\n", strconv.Quote(endpointMethod(endpoint, defaultMethod)), pathExpr)
}

func emitSwiftBridgeClient(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	emitSwiftBridgeModels(buf, bridges)
	if len(bridges) > 0 {
		fmt.Fprintln(buf)
	}
	fmt.Fprintln(buf, "public final class GSXGeneratedBridgeClient {")
	fmt.Fprintln(buf, "    private let client: GSXBridgeClient")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public init(client: GSXBridgeClient) {")
	fmt.Fprintln(buf, "        self.client = client")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(dataClient: GSXDataClient) {")
	fmt.Fprintln(buf, "        self.init(client: GSXBridgeClient(dataClient: dataClient))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(transport: any GSXTransport) {")
	fmt.Fprintln(buf, "        self.init(dataClient: GSXDataClient(transport: transport))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:]) {")
	fmt.Fprintln(buf, "        self.init(dataClient: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {")
	fmt.Fprintln(buf, "        self.init(dataClient: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {")
	fmt.Fprintln(buf, "        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {")
	fmt.Fprintln(buf, "        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))")
	fmt.Fprintln(buf, "    }")
	for _, bridge := range bridges {
		fmt.Fprintln(buf)
		resultType := "GSXResponse"
		outputModel := swiftBridgeModelName(bridge, "Response")
		if len(bridge.Output) > 0 {
			resultType = outputModel
		}
		fmt.Fprintf(buf, "    public func %s(%s) async throws -> %s {\n", swiftBridgeMethodName(bridge), swiftParamList(bridge.Input), resultType)
		emitSwiftBridgeRequest(buf, bridge)
		fmt.Fprintf(buf, "        let response = try await client.call(request, policy: %s)\n", swiftBridgeRequestPolicy(bridge))
		if len(bridge.Output) > 0 {
			fmt.Fprintf(buf, "        return try response.decodedJSON(%s.self)\n", outputModel)
		} else {
			fmt.Fprintln(buf, "        return response")
		}
		fmt.Fprintln(buf, "    }")
	}
	fmt.Fprintln(buf, "}")
}

func emitSwiftBridgeModels(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	for _, bridge := range bridges {
		if len(bridge.Input) > 0 {
			emitSwiftEndpointModel(buf, swiftBridgeModelName(bridge, "Input"), bridge.Input)
			fmt.Fprintln(buf)
		}
		if len(bridge.Output) > 0 {
			emitSwiftEndpointModel(buf, swiftBridgeModelName(bridge, "Response"), bridge.Output)
			fmt.Fprintln(buf)
		}
	}
}

func emitSwiftBridgeRequest(buf *bytes.Buffer, bridge bridgeDeclaration) {
	if len(bridge.Input) > 0 {
		fmt.Fprintf(buf, "        let input = %s(%s)\n", swiftBridgeModelName(bridge, "Input"), swiftModelInitArgs(bridge.Input))
		fmt.Fprintf(buf, "        let request = try GSXRequest.json(method: \"POST\", path: %s, body: input)\n", strconv.Quote(bridge.Path))
		return
	}
	fmt.Fprintf(buf, "        let request = GSXRequest(method: \"POST\", path: %s)\n", strconv.Quote(bridge.Path))
}

func emitKotlinDeclarations(cfg *projectConfig) []byte {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "// Code generated by gsxnative. DO NOT EDIT.")
	fmt.Fprintln(&buf, "package generated")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXAuthRequirement")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXBridgeClient")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXCapabilityNegotiator")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXCapabilityReport")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXCapabilitySpec")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXDataClient")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXHTTPTransport")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXRequest")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXRequestPolicy")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXResponse")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXRoute")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXTokenStore")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXTransport")
	fmt.Fprintln(&buf, "import org.json.JSONObject")
	fmt.Fprintln(&buf)
	emitKotlinGeneratedSpecs(&buf)
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "data class GSXGeneratedRouteSpec(")
	fmt.Fprintln(&buf, "    val name: String,")
	fmt.Fprintln(&buf, "    val path: String,")
	fmt.Fprintln(&buf, "    val component: String,")
	fmt.Fprintln(&buf, "    val params: List<String> = emptyList(),")
	fmt.Fprintln(&buf, "    val auth: GSXAuthRequirement = GSXAuthRequirement.Optional,")
	fmt.Fprintln(&buf, ")")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "object GSXRoutes {")
	fmt.Fprintln(&buf, "    val specs: List<GSXGeneratedRouteSpec> = listOf(")
	for _, route := range cfg.Routes {
		fmt.Fprintf(&buf, "        GSXGeneratedRouteSpec(name = %s, path = %s, component = %s, params = %s, auth = %s),\n",
			strconv.Quote(route.Name), strconv.Quote(route.Path), strconv.Quote(route.Component), kotlinStringList(paramNames(route.Params)), kotlinAuthRequirement(route.Auth))
	}
	fmt.Fprintln(&buf, "    )")
	if len(cfg.Routes) > 0 {
		fmt.Fprintln(&buf)
	}
	for _, route := range cfg.Routes {
		emitKotlinRoute(&buf, route)
	}
	fmt.Fprintln(&buf, "}")
	fmt.Fprintln(&buf)
	emitKotlinEndpointSpecs(&buf, "GSXDataLoaders", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitKotlinEndpointSpecs(&buf, "GSXActions", cfg.Actions, "POST")
	fmt.Fprintln(&buf)
	emitKotlinCapabilitySpecs(&buf, cfg.Capabilities)
	fmt.Fprintln(&buf)
	emitKotlinBridgeSpecs(&buf, cfg.Bridges)
	fmt.Fprintln(&buf)
	emitKotlinEndpointClient(&buf, "GSXGeneratedDataClient", "load", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitKotlinEndpointClient(&buf, "GSXGeneratedActionClient", "submit", cfg.Actions, "POST")
	fmt.Fprintln(&buf)
	emitKotlinBridgeClient(&buf, cfg.Bridges)
	return buf.Bytes()
}

func emitKotlinGeneratedSpecs(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "data class GSXGeneratedParamSpec(")
	fmt.Fprintln(buf, "    val name: String,")
	fmt.Fprintln(buf, "    val type: String,")
	fmt.Fprintln(buf, ")")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "data class GSXGeneratedEndpointSpec(")
	fmt.Fprintln(buf, "    val name: String,")
	fmt.Fprintln(buf, "    val method: String,")
	fmt.Fprintln(buf, "    val path: String,")
	fmt.Fprintln(buf, "    val params: List<GSXGeneratedParamSpec> = emptyList(),")
	fmt.Fprintln(buf, "    val input: List<GSXGeneratedParamSpec> = emptyList(),")
	fmt.Fprintln(buf, "    val output: List<GSXGeneratedParamSpec> = emptyList(),")
	fmt.Fprintln(buf, "    val cacheTTLSeconds: Int? = null,")
	fmt.Fprintln(buf, "    val invalidates: List<String> = emptyList(),")
	fmt.Fprintln(buf, "    val optimistic: String? = null,")
	fmt.Fprintln(buf, "    val auth: GSXAuthRequirement = GSXAuthRequirement.Optional,")
	fmt.Fprintln(buf, "    val retryAttempts: Int? = null,")
	fmt.Fprintln(buf, ")")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "data class GSXGeneratedCapabilitySpec(")
	fmt.Fprintln(buf, "    val name: String,")
	fmt.Fprintln(buf, "    val targets: List<String> = emptyList(),")
	fmt.Fprintln(buf, "    val required: Boolean = false,")
	fmt.Fprintln(buf, ")")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "data class GSXGeneratedBridgeSpec(")
	fmt.Fprintln(buf, "    val service: String,")
	fmt.Fprintln(buf, "    val method: String,")
	fmt.Fprintln(buf, "    val path: String,")
	fmt.Fprintln(buf, "    val input: List<GSXGeneratedParamSpec> = emptyList(),")
	fmt.Fprintln(buf, "    val output: List<GSXGeneratedParamSpec> = emptyList(),")
	fmt.Fprintln(buf, "    val auth: GSXAuthRequirement = GSXAuthRequirement.Optional,")
	fmt.Fprintln(buf, "    val retryAttempts: Int? = null,")
	fmt.Fprintln(buf, ")")
}

func emitKotlinRoute(buf *bytes.Buffer, route routeDeclaration) {
	name := kotlinIdentifier(route.Name)
	if len(route.Params) == 0 {
		fmt.Fprintf(buf, "    val %s: GSXRoute = GSXRoute(%s, auth = %s)\n", name, strconv.Quote(route.Name), kotlinAuthRequirement(route.Auth))
		return
	}
	fmt.Fprintf(buf, "    fun %s(%s): GSXRoute = GSXRoute(\n", name, kotlinParamList(route.Params))
	fmt.Fprintf(buf, "        %s,\n", strconv.Quote(route.Name))
	fmt.Fprintln(buf, "        mapOf(")
	for _, param := range route.Params {
		fmt.Fprintf(buf, "            %s to %s,\n", strconv.Quote(param.Name), kotlinStringValue(param.Name, param.Type))
	}
	fmt.Fprintln(buf, "        ),")
	fmt.Fprintf(buf, "        auth = %s,\n", kotlinAuthRequirement(route.Auth))
	fmt.Fprintln(buf, "    )")
}

func emitKotlinEndpointSpecs(buf *bytes.Buffer, objectName string, endpoints []endpointDeclaration, defaultMethod string) {
	fmt.Fprintf(buf, "object %s {\n", objectName)
	fmt.Fprintln(buf, "    val specs: List<GSXGeneratedEndpointSpec> = listOf(")
	for _, endpoint := range endpoints {
		fmt.Fprintf(buf, "        GSXGeneratedEndpointSpec(name = %s, method = %s, path = %s, params = %s, input = %s, output = %s, cacheTTLSeconds = %s, invalidates = %s, optimistic = %s, auth = %s, retryAttempts = %s),\n",
			strconv.Quote(endpoint.Name),
			strconv.Quote(endpointMethod(endpoint, defaultMethod)),
			strconv.Quote(endpoint.Path),
			kotlinParamSpecList(endpoint.Params),
			kotlinParamSpecList(endpoint.Input),
			kotlinParamSpecList(endpoint.Output),
			kotlinOptionalInt(endpoint.CacheTTLSeconds),
			kotlinStringList(endpoint.Invalidates),
			kotlinOptionalString(endpoint.Optimistic),
			kotlinAuthRequirement(endpoint.Auth),
			kotlinOptionalInt(endpoint.RetryAttempts),
		)
	}
	fmt.Fprintln(buf, "    )")
	fmt.Fprintln(buf, "}")
}

func emitKotlinCapabilitySpecs(buf *bytes.Buffer, capabilities []capabilityDeclaration) {
	fmt.Fprintln(buf, "object GSXCapabilities {")
	fmt.Fprintln(buf, "    val specs: List<GSXGeneratedCapabilitySpec> = listOf(")
	for _, capability := range capabilities {
		fmt.Fprintf(buf, "        GSXGeneratedCapabilitySpec(name = %s, targets = %s, required = %s),\n",
			strconv.Quote(capability.Name),
			kotlinStringList(capabilityTargets(capability)),
			kotlinBool(capability.Required),
		)
	}
	fmt.Fprintln(buf, "    )")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    val runtimeSpecs: List<GSXCapabilitySpec> = specs.map { spec ->")
	fmt.Fprintln(buf, "        GSXCapabilitySpec(name = spec.name, targets = spec.targets, required = spec.required)")
	fmt.Fprintln(buf, "    }")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    suspend fun negotiate(")
	fmt.Fprintln(buf, "        negotiator: GSXCapabilityNegotiator,")
	fmt.Fprintln(buf, "        target: String = \"android\",")
	fmt.Fprintln(buf, "        path: String = \"/api/capabilities\",")
	fmt.Fprintln(buf, "    ): GSXCapabilityReport = negotiator.negotiate(required = runtimeSpecs, target = target, path = path)")
	fmt.Fprintln(buf, "}")
}

func emitKotlinBridgeSpecs(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	fmt.Fprintln(buf, "object GSXBridges {")
	fmt.Fprintln(buf, "    val specs: List<GSXGeneratedBridgeSpec> = listOf(")
	for _, bridge := range bridges {
		fmt.Fprintf(buf, "        GSXGeneratedBridgeSpec(service = %s, method = %s, path = %s, input = %s, output = %s, auth = %s, retryAttempts = %s),\n",
			strconv.Quote(bridge.Service),
			strconv.Quote(bridge.Method),
			strconv.Quote(bridge.Path),
			kotlinParamSpecList(bridge.Input),
			kotlinParamSpecList(bridge.Output),
			kotlinAuthRequirement(bridge.Auth),
			kotlinOptionalInt(bridge.RetryAttempts),
		)
	}
	fmt.Fprintln(buf, "    )")
	fmt.Fprintln(buf, "}")
}

func emitKotlinEndpointClient(buf *bytes.Buffer, className, operation string, endpoints []endpointDeclaration, defaultMethod string) {
	emitKotlinEndpointModels(buf, className, endpoints)
	if len(endpoints) > 0 {
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "class %s(\n", className)
	fmt.Fprintln(buf, "    private val client: GSXDataClient,")
	fmt.Fprintln(buf, ") {")
	fmt.Fprintln(buf, "    constructor(transport: GSXTransport) : this(GSXDataClient(transport))")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(")
	fmt.Fprintln(buf, "        GSXDataClient(GSXHTTPTransport(baseURL = baseURL, defaultHeaders = defaultHeaders)),")
	fmt.Fprintln(buf, "    )")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore) : this(")
	fmt.Fprintln(buf, "        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders, tokenStore = tokenStore),")
	fmt.Fprintln(buf, "    )")
	for _, endpoint := range endpoints {
		fmt.Fprintln(buf)
		resultType := "GSXResponse"
		outputModel := kotlinEndpointModelName(className, endpoint, "Response")
		if len(endpoint.Output) > 0 {
			resultType = outputModel
		}
		fmt.Fprintf(buf, "    suspend fun %s(%s): %s {\n", kotlinIdentifier(endpoint.Name), kotlinEndpointParamList(endpoint), resultType)
		emitKotlinEndpointRequest(buf, className, endpoint, defaultMethod)
		fmt.Fprintf(buf, "        val response = client.%s(request, policy = %s)\n", operation, kotlinRequestPolicy(endpoint))
		if len(endpoint.Output) > 0 {
			fmt.Fprintf(buf, "        return %s.fromJSON(response.text())\n", outputModel)
		} else {
			fmt.Fprintln(buf, "        return response")
		}
		fmt.Fprintln(buf, "    }")
	}
	fmt.Fprintln(buf, "}")
}

func emitKotlinEndpointModels(buf *bytes.Buffer, className string, endpoints []endpointDeclaration) {
	for _, endpoint := range endpoints {
		if len(endpoint.Input) > 0 {
			emitKotlinEndpointInputModel(buf, kotlinEndpointModelName(className, endpoint, "Input"), endpoint.Input)
			fmt.Fprintln(buf)
		}
		if len(endpoint.Output) > 0 {
			emitKotlinEndpointOutputModel(buf, kotlinEndpointModelName(className, endpoint, "Response"), endpoint.Output)
			fmt.Fprintln(buf)
		}
	}
}

func emitKotlinEndpointInputModel(buf *bytes.Buffer, name string, fields []paramDeclaration) {
	fmt.Fprintf(buf, "data class %s(\n", name)
	for _, field := range fields {
		fmt.Fprintf(buf, "    val %s: %s,\n", kotlinIdentifier(field.Name), kotlinTypeForDecl(field.Type))
	}
	fmt.Fprintln(buf, ") {")
	emitKotlinModelToJSON(buf, fields)
	fmt.Fprintln(buf)
	emitKotlinModelFromJSON(buf, name, fields)
	fmt.Fprintln(buf, "}")
}

func emitKotlinEndpointOutputModel(buf *bytes.Buffer, name string, fields []paramDeclaration) {
	fmt.Fprintf(buf, "data class %s(\n", name)
	for _, field := range fields {
		fmt.Fprintf(buf, "    val %s: %s,\n", kotlinIdentifier(field.Name), kotlinTypeForDecl(field.Type))
	}
	fmt.Fprintln(buf, ") {")
	emitKotlinModelToJSON(buf, fields)
	fmt.Fprintln(buf)
	emitKotlinModelFromJSON(buf, name, fields)
	fmt.Fprintln(buf, "}")
}

func emitKotlinModelToJSON(buf *bytes.Buffer, fields []paramDeclaration) {
	fmt.Fprintln(buf, "    fun toJSON(): String {")
	fmt.Fprintln(buf, "        val objectValue = JSONObject()")
	for _, field := range fields {
		fmt.Fprintf(buf, "        objectValue.put(%s, %s)\n", strconv.Quote(field.Name), kotlinIdentifier(field.Name))
	}
	fmt.Fprintln(buf, "        return objectValue.toString()")
	fmt.Fprintln(buf, "    }")
}

func emitKotlinModelFromJSON(buf *bytes.Buffer, name string, fields []paramDeclaration) {
	fmt.Fprintln(buf, "    companion object {")
	fmt.Fprintln(buf, "        fun fromJSON(json: String): "+name+" {")
	fmt.Fprintln(buf, "            val objectValue = JSONObject(json)")
	fmt.Fprintf(buf, "            return %s(\n", name)
	for _, field := range fields {
		fmt.Fprintf(buf, "                %s = %s,\n", kotlinIdentifier(field.Name), kotlinJSONGetter(field))
	}
	fmt.Fprintln(buf, "            )")
	fmt.Fprintln(buf, "        }")
	fmt.Fprintln(buf, "    }")
}

func emitKotlinEndpointRequest(buf *bytes.Buffer, className string, endpoint endpointDeclaration, defaultMethod string) {
	pathExpr := strconv.Quote(endpoint.Path)
	if len(endpoint.Params) > 0 {
		fmt.Fprintf(buf, "        val path = GSXRequest.resolvedPath(%s, params = %s)\n", strconv.Quote(endpoint.Path), kotlinParamMap(endpoint.Params))
		pathExpr = "path"
	}
	if len(endpoint.Input) > 0 {
		fmt.Fprintf(buf, "        val input = %s(%s)\n", kotlinEndpointModelName(className, endpoint, "Input"), kotlinModelInitArgs(endpoint.Input))
		fmt.Fprintf(buf, "        val request = GSXRequest.json(method = %s, path = %s, json = input.toJSON())\n", strconv.Quote(endpointMethod(endpoint, defaultMethod)), pathExpr)
		return
	}
	fmt.Fprintf(buf, "        val request = GSXRequest(method = %s, path = %s)\n", strconv.Quote(endpointMethod(endpoint, defaultMethod)), pathExpr)
}

func emitKotlinBridgeClient(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	emitKotlinBridgeModels(buf, bridges)
	if len(bridges) > 0 {
		fmt.Fprintln(buf)
	}
	fmt.Fprintln(buf, "class GSXGeneratedBridgeClient(")
	fmt.Fprintln(buf, "    private val client: GSXBridgeClient,")
	fmt.Fprintln(buf, ") {")
	fmt.Fprintln(buf, "    constructor(dataClient: GSXDataClient) : this(GSXBridgeClient(dataClient))")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    constructor(transport: GSXTransport) : this(GSXDataClient(transport))")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(")
	fmt.Fprintln(buf, "        GSXDataClient(GSXHTTPTransport(baseURL = baseURL, defaultHeaders = defaultHeaders)),")
	fmt.Fprintln(buf, "    )")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore) : this(")
	fmt.Fprintln(buf, "        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders, tokenStore = tokenStore),")
	fmt.Fprintln(buf, "    )")
	for _, bridge := range bridges {
		fmt.Fprintln(buf)
		resultType := "GSXResponse"
		outputModel := kotlinBridgeModelName(bridge, "Response")
		if len(bridge.Output) > 0 {
			resultType = outputModel
		}
		fmt.Fprintf(buf, "    suspend fun %s(%s): %s {\n", kotlinBridgeMethodName(bridge), kotlinParamList(bridge.Input), resultType)
		emitKotlinBridgeRequest(buf, bridge)
		fmt.Fprintf(buf, "        val response = client.call(request, policy = %s)\n", kotlinBridgeRequestPolicy(bridge))
		if len(bridge.Output) > 0 {
			fmt.Fprintf(buf, "        return %s.fromJSON(response.text())\n", outputModel)
		} else {
			fmt.Fprintln(buf, "        return response")
		}
		fmt.Fprintln(buf, "    }")
	}
	fmt.Fprintln(buf, "}")
}

func emitKotlinBridgeModels(buf *bytes.Buffer, bridges []bridgeDeclaration) {
	for _, bridge := range bridges {
		if len(bridge.Input) > 0 {
			emitKotlinEndpointInputModel(buf, kotlinBridgeModelName(bridge, "Input"), bridge.Input)
			fmt.Fprintln(buf)
		}
		if len(bridge.Output) > 0 {
			emitKotlinEndpointOutputModel(buf, kotlinBridgeModelName(bridge, "Response"), bridge.Output)
			fmt.Fprintln(buf)
		}
	}
}

func emitKotlinBridgeRequest(buf *bytes.Buffer, bridge bridgeDeclaration) {
	if len(bridge.Input) > 0 {
		fmt.Fprintf(buf, "        val input = %s(%s)\n", kotlinBridgeModelName(bridge, "Input"), kotlinModelInitArgs(bridge.Input))
		fmt.Fprintf(buf, "        val request = GSXRequest.json(method = \"POST\", path = %s, json = input.toJSON())\n", strconv.Quote(bridge.Path))
		return
	}
	fmt.Fprintf(buf, "        val request = GSXRequest(method = \"POST\", path = %s)\n", strconv.Quote(bridge.Path))
}

func endpointMethod(endpoint endpointDeclaration, fallback string) string {
	method := strings.ToUpper(strings.TrimSpace(endpoint.Method))
	if method == "" {
		return fallback
	}
	return method
}

func bridgeName(bridge bridgeDeclaration) string {
	return bridge.Service + "." + bridge.Method
}

func capabilityTargets(capability capabilityDeclaration) []string {
	if len(capability.Targets) == 0 {
		return []string{"ios", "android"}
	}
	targets := make([]string, 0, len(capability.Targets))
	for _, tgt := range capability.Targets {
		tgt = strings.ToLower(strings.TrimSpace(tgt))
		if tgt != "" {
			targets = append(targets, tgt)
		}
	}
	return targets
}

func paramNames(params []paramDeclaration) []string {
	names := make([]string, 0, len(params))
	for _, param := range params {
		names = append(names, param.Name)
	}
	sort.Strings(names)
	return names
}

func swiftParamList(params []paramDeclaration) string {
	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%s: %s", swiftIdentifier(param.Name), swiftTypeForDecl(param.Type)))
	}
	return strings.Join(parts, ", ")
}

func kotlinParamList(params []paramDeclaration) string {
	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%s: %s", kotlinIdentifier(param.Name), kotlinTypeForDecl(param.Type)))
	}
	return strings.Join(parts, ", ")
}

func swiftStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func kotlinStringList(values []string) string {
	if len(values) == 0 {
		return "emptyList()"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "listOf(" + strings.Join(quoted, ", ") + ")"
}

func swiftBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func kotlinBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func swiftParamSpecArray(params []paramDeclaration) string {
	if len(params) == 0 {
		return "[]"
	}
	specs := make([]string, 0, len(params))
	for _, param := range params {
		specs = append(specs, fmt.Sprintf("GSXGeneratedParamSpec(name: %s, type: %s)", strconv.Quote(param.Name), strconv.Quote(normalizedDeclarationType(param.Type))))
	}
	return "[" + strings.Join(specs, ", ") + "]"
}

func kotlinParamSpecList(params []paramDeclaration) string {
	if len(params) == 0 {
		return "emptyList()"
	}
	specs := make([]string, 0, len(params))
	for _, param := range params {
		specs = append(specs, fmt.Sprintf("GSXGeneratedParamSpec(name = %s, type = %s)", strconv.Quote(param.Name), strconv.Quote(normalizedDeclarationType(param.Type))))
	}
	return "listOf(" + strings.Join(specs, ", ") + ")"
}

func normalizedDeclarationType(typ string) string {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		return "string"
	}
	if typ == "boolean" {
		return "bool"
	}
	return typ
}

func swiftOptionalInt(value int) string {
	if value <= 0 {
		return "nil"
	}
	return strconv.Itoa(value)
}

func kotlinOptionalInt(value int) string {
	if value <= 0 {
		return "null"
	}
	return strconv.Itoa(value)
}

func swiftOptionalString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "nil"
	}
	return strconv.Quote(value)
}

func kotlinOptionalString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return strconv.Quote(value)
}

func swiftAuthRequirement(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "required":
		return "GSXAuthRequirement.required"
	case "none":
		return "GSXAuthRequirement.none"
	default:
		return "GSXAuthRequirement.optional"
	}
}

func kotlinAuthRequirement(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "required":
		return "GSXAuthRequirement.Required"
	case "none":
		return "GSXAuthRequirement.None"
	default:
		return "GSXAuthRequirement.Optional"
	}
}

func swiftEndpointModelName(className string, endpoint endpointDeclaration, suffix string) string {
	return endpointModelName(className, endpoint, suffix)
}

func kotlinEndpointModelName(className string, endpoint endpointDeclaration, suffix string) string {
	return endpointModelName(className, endpoint, suffix)
}

func endpointModelName(className string, endpoint endpointDeclaration, suffix string) string {
	prefix := strings.TrimPrefix(className, "GSXGenerated")
	prefix = strings.TrimSuffix(prefix, "Client")
	if prefix == "" {
		prefix = "Endpoint"
	}
	return "GSXGenerated" + prefix + pascalIdentifier(endpoint.Name, "Endpoint") + suffix
}

func swiftBridgeModelName(bridge bridgeDeclaration, suffix string) string {
	return bridgeModelName(bridge, suffix)
}

func kotlinBridgeModelName(bridge bridgeDeclaration, suffix string) string {
	return bridgeModelName(bridge, suffix)
}

func bridgeModelName(bridge bridgeDeclaration, suffix string) string {
	return "GSXGeneratedBridge" + pascalIdentifier(bridge.Service, "Service") + pascalIdentifier(bridge.Method, "Method") + suffix
}

func swiftBridgeMethodName(bridge bridgeDeclaration) string {
	return lowerIdentifier(bridge.Service + " " + bridge.Method)
}

func kotlinBridgeMethodName(bridge bridgeDeclaration) string {
	return lowerIdentifier(bridge.Service + " " + bridge.Method)
}

func swiftEndpointParamList(endpoint endpointDeclaration) string {
	return swiftParamList(endpointMethodParams(endpoint))
}

func kotlinEndpointParamList(endpoint endpointDeclaration) string {
	return kotlinParamList(endpointMethodParams(endpoint))
}

func endpointMethodParams(endpoint endpointDeclaration) []paramDeclaration {
	params := make([]paramDeclaration, 0, len(endpoint.Params)+len(endpoint.Input))
	params = append(params, endpoint.Params...)
	params = append(params, endpoint.Input...)
	return params
}

func swiftParamMap(params []paramDeclaration) string {
	if len(params) == 0 {
		return "[:]"
	}
	entries := make([]string, 0, len(params))
	for _, param := range params {
		entries = append(entries, fmt.Sprintf("%s: %s", strconv.Quote(param.Name), swiftStringValue(param.Name, param.Type)))
	}
	return "[" + strings.Join(entries, ", ") + "]"
}

func kotlinParamMap(params []paramDeclaration) string {
	if len(params) == 0 {
		return "emptyMap()"
	}
	entries := make([]string, 0, len(params))
	for _, param := range params {
		entries = append(entries, fmt.Sprintf("%s to %s", strconv.Quote(param.Name), kotlinStringValue(param.Name, param.Type)))
	}
	return "mapOf(" + strings.Join(entries, ", ") + ")"
}

func swiftModelInitArgs(fields []paramDeclaration) string {
	args := make([]string, 0, len(fields))
	for _, field := range fields {
		args = append(args, fmt.Sprintf("%s: %s", swiftIdentifier(field.Name), swiftIdentifier(field.Name)))
	}
	return strings.Join(args, ", ")
}

func kotlinModelInitArgs(fields []paramDeclaration) string {
	args := make([]string, 0, len(fields))
	for _, field := range fields {
		args = append(args, fmt.Sprintf("%s = %s", kotlinIdentifier(field.Name), kotlinIdentifier(field.Name)))
	}
	return strings.Join(args, ", ")
}

func swiftRequestPolicy(endpoint endpointDeclaration) string {
	parts := []string{fmt.Sprintf("name: %s", strconv.Quote(endpoint.Name))}
	if endpoint.CacheTTLSeconds > 0 {
		parts = append(parts, fmt.Sprintf("cacheTTLSeconds: %d", endpoint.CacheTTLSeconds))
	}
	if len(endpoint.Invalidates) > 0 {
		parts = append(parts, "invalidates: "+swiftStringArray(endpoint.Invalidates))
	}
	if endpoint.Optimistic != "" {
		parts = append(parts, "optimistic: "+strconv.Quote(endpoint.Optimistic))
	}
	if endpoint.Auth != "" {
		parts = append(parts, "auth: "+swiftAuthRequirement(endpoint.Auth))
	}
	if endpoint.RetryAttempts > 0 {
		parts = append(parts, fmt.Sprintf("retryAttempts: %d", endpoint.RetryAttempts))
	}
	return "GSXRequestPolicy(" + strings.Join(parts, ", ") + ")"
}

func swiftBridgeRequestPolicy(bridge bridgeDeclaration) string {
	parts := []string{fmt.Sprintf("name: %s", strconv.Quote(bridgeName(bridge)))}
	if bridge.Auth != "" {
		parts = append(parts, "auth: "+swiftAuthRequirement(bridge.Auth))
	}
	if bridge.RetryAttempts > 0 {
		parts = append(parts, fmt.Sprintf("retryAttempts: %d", bridge.RetryAttempts))
	}
	return "GSXRequestPolicy(" + strings.Join(parts, ", ") + ")"
}

func kotlinRequestPolicy(endpoint endpointDeclaration) string {
	parts := []string{fmt.Sprintf("name = %s", strconv.Quote(endpoint.Name))}
	if endpoint.CacheTTLSeconds > 0 {
		parts = append(parts, fmt.Sprintf("cacheTTLSeconds = %d", endpoint.CacheTTLSeconds))
	}
	if len(endpoint.Invalidates) > 0 {
		parts = append(parts, "invalidates = "+kotlinStringList(endpoint.Invalidates))
	}
	if endpoint.Optimistic != "" {
		parts = append(parts, "optimistic = "+strconv.Quote(endpoint.Optimistic))
	}
	if endpoint.Auth != "" {
		parts = append(parts, "auth = "+kotlinAuthRequirement(endpoint.Auth))
	}
	if endpoint.RetryAttempts > 0 {
		parts = append(parts, fmt.Sprintf("retryAttempts = %d", endpoint.RetryAttempts))
	}
	return "GSXRequestPolicy(" + strings.Join(parts, ", ") + ")"
}

func kotlinBridgeRequestPolicy(bridge bridgeDeclaration) string {
	parts := []string{fmt.Sprintf("name = %s", strconv.Quote(bridgeName(bridge)))}
	if bridge.Auth != "" {
		parts = append(parts, "auth = "+kotlinAuthRequirement(bridge.Auth))
	}
	if bridge.RetryAttempts > 0 {
		parts = append(parts, fmt.Sprintf("retryAttempts = %d", bridge.RetryAttempts))
	}
	return "GSXRequestPolicy(" + strings.Join(parts, ", ") + ")"
}

func kotlinJSONGetter(field paramDeclaration) string {
	name := strconv.Quote(field.Name)
	switch strings.ToLower(strings.TrimSpace(field.Type)) {
	case "int":
		return "objectValue.getInt(" + name + ")"
	case "double", "float":
		return "objectValue.getDouble(" + name + ")"
	case "bool", "boolean":
		return "objectValue.getBoolean(" + name + ")"
	default:
		return "objectValue.getString(" + name + ")"
	}
}

func swiftStringValue(name, typ string) string {
	name = swiftIdentifier(name)
	if strings.EqualFold(typ, "string") || strings.TrimSpace(typ) == "" {
		return name
	}
	return "String(" + name + ")"
}

func kotlinStringValue(name, typ string) string {
	name = kotlinIdentifier(name)
	if strings.EqualFold(typ, "string") || strings.TrimSpace(typ) == "" {
		return name
	}
	return name + ".toString()"
}

func swiftTypeForDecl(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "int":
		return "Int"
	case "double", "float":
		return "Double"
	case "bool", "boolean":
		return "Bool"
	default:
		return "String"
	}
}

func kotlinTypeForDecl(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "int":
		return "Int"
	case "double", "float":
		return "Double"
	case "bool", "boolean":
		return "Boolean"
	default:
		return "String"
	}
}

func swiftIdentifier(value string) string {
	return lowerIdentifier(value)
}

func kotlinIdentifier(value string) string {
	return lowerIdentifier(value)
}

func lowerIdentifier(value string) string {
	name := pascalIdentifier(value, "Generated")
	runes := []rune(name)
	if len(runes) == 0 {
		return "generated"
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
