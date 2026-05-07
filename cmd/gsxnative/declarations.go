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
	"unicode"

	"github.com/odvcencio/gosx-native/target"
	"github.com/odvcencio/gosx/nir"
)

type sourceDeclarations struct {
	Routes      []routeDeclaration
	DataLoaders []endpointDeclaration
	Actions     []endpointDeclaration
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
	return validateEndpoints("action", cfg.Actions)
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
	return &clone
}

func (d sourceDeclarations) hasAny() bool {
	return d.hasRoutes() || d.hasDataLoaders() || d.hasActions()
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
	if kind != "route" && kind != "data" && kind != "action" {
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
	if err := rejectUnknownDeclarationFields("route", fields, "name", "path", "component", "params", "param"); err != nil {
		return routeDeclaration{}, err
	}
	params, err := parseSourceParams(append(fields["params"], fields["param"]...))
	if err != nil {
		return routeDeclaration{}, err
	}
	return routeDeclaration{
		Name:      firstField(fields, "name"),
		Path:      firstField(fields, "path"),
		Component: firstField(fields, "component"),
		Params:    params,
	}, nil
}

func sourceEndpointDeclaration(kind string, fields map[string][]string) (endpointDeclaration, error) {
	if err := requireDeclarationFields(kind, fields, "name", "method", "path"); err != nil {
		return endpointDeclaration{}, err
	}
	if err := rejectUnknownDeclarationFields(kind, fields, "name", "method", "path"); err != nil {
		return endpointDeclaration{}, err
	}
	return endpointDeclaration{
		Name:   firstField(fields, "name"),
		Method: firstField(fields, "method"),
		Path:   firstField(fields, "path"),
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
	fmt.Fprintln(&buf, "public struct GSXGeneratedRouteSpec: Equatable {")
	fmt.Fprintln(&buf, "    public let name: String")
	fmt.Fprintln(&buf, "    public let path: String")
	fmt.Fprintln(&buf, "    public let component: String")
	fmt.Fprintln(&buf, "    public let params: [String]")
	fmt.Fprintln(&buf, "}")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "public enum GSXRoutes {")
	fmt.Fprintln(&buf, "    public static let specs: [GSXGeneratedRouteSpec] = [")
	for _, route := range cfg.Routes {
		fmt.Fprintf(&buf, "        GSXGeneratedRouteSpec(name: %s, path: %s, component: %s, params: %s),\n",
			strconv.Quote(route.Name), strconv.Quote(route.Path), strconv.Quote(route.Component), swiftStringArray(paramNames(route.Params)))
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
	emitSwiftEndpointClient(&buf, "GSXGeneratedDataClient", "load", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitSwiftEndpointClient(&buf, "GSXGeneratedActionClient", "submit", cfg.Actions, "POST")
	return buf.Bytes()
}

func emitSwiftRoute(buf *bytes.Buffer, route routeDeclaration) {
	name := swiftIdentifier(route.Name)
	if len(route.Params) == 0 {
		fmt.Fprintf(buf, "    public static let %s = GSXRoute(%s)\n", name, strconv.Quote(route.Name))
		return
	}
	fmt.Fprintf(buf, "    public static func %s(%s) -> GSXRoute {\n", name, swiftParamList(route.Params))
	fmt.Fprintln(buf, "        GSXRoute(")
	fmt.Fprintf(buf, "            %s,\n", strconv.Quote(route.Name))
	fmt.Fprintln(buf, "            params: [")
	for _, param := range route.Params {
		fmt.Fprintf(buf, "                %s: %s,\n", strconv.Quote(param.Name), swiftStringValue(param.Name, param.Type))
	}
	fmt.Fprintln(buf, "            ]")
	fmt.Fprintln(buf, "        )")
	fmt.Fprintln(buf, "    }")
}

func emitSwiftEndpointClient(buf *bytes.Buffer, className, operation string, endpoints []endpointDeclaration, defaultMethod string) {
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
		fmt.Fprintf(buf, "    public func %s() async throws -> GSXResponse {\n", swiftIdentifier(endpoint.Name))
		fmt.Fprintf(buf, "        try await client.%s(GSXRequest(method: %s, path: %s))\n",
			operation, strconv.Quote(endpointMethod(endpoint, defaultMethod)), strconv.Quote(endpoint.Path))
		fmt.Fprintln(buf, "    }")
	}
	fmt.Fprintln(buf, "}")
}

func emitKotlinDeclarations(cfg *projectConfig) []byte {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "// Code generated by gsxnative. DO NOT EDIT.")
	fmt.Fprintln(&buf, "package generated")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXDataClient")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXHTTPTransport")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXRequest")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXResponse")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXRoute")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXTokenStore")
	fmt.Fprintln(&buf, "import com.gosx.nativekit.GSXTransport")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "data class GSXGeneratedRouteSpec(")
	fmt.Fprintln(&buf, "    val name: String,")
	fmt.Fprintln(&buf, "    val path: String,")
	fmt.Fprintln(&buf, "    val component: String,")
	fmt.Fprintln(&buf, "    val params: List<String> = emptyList(),")
	fmt.Fprintln(&buf, ")")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "object GSXRoutes {")
	fmt.Fprintln(&buf, "    val specs: List<GSXGeneratedRouteSpec> = listOf(")
	for _, route := range cfg.Routes {
		fmt.Fprintf(&buf, "        GSXGeneratedRouteSpec(name = %s, path = %s, component = %s, params = %s),\n",
			strconv.Quote(route.Name), strconv.Quote(route.Path), strconv.Quote(route.Component), kotlinStringList(paramNames(route.Params)))
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
	emitKotlinEndpointClient(&buf, "GSXGeneratedDataClient", "load", cfg.DataLoaders, "GET")
	fmt.Fprintln(&buf)
	emitKotlinEndpointClient(&buf, "GSXGeneratedActionClient", "submit", cfg.Actions, "POST")
	return buf.Bytes()
}

func emitKotlinRoute(buf *bytes.Buffer, route routeDeclaration) {
	name := kotlinIdentifier(route.Name)
	if len(route.Params) == 0 {
		fmt.Fprintf(buf, "    val %s: GSXRoute = GSXRoute(%s)\n", name, strconv.Quote(route.Name))
		return
	}
	fmt.Fprintf(buf, "    fun %s(%s): GSXRoute = GSXRoute(\n", name, kotlinParamList(route.Params))
	fmt.Fprintf(buf, "        %s,\n", strconv.Quote(route.Name))
	fmt.Fprintln(buf, "        mapOf(")
	for _, param := range route.Params {
		fmt.Fprintf(buf, "            %s to %s,\n", strconv.Quote(param.Name), kotlinStringValue(param.Name, param.Type))
	}
	fmt.Fprintln(buf, "        ),")
	fmt.Fprintln(buf, "    )")
}

func emitKotlinEndpointClient(buf *bytes.Buffer, className, operation string, endpoints []endpointDeclaration, defaultMethod string) {
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
		fmt.Fprintf(buf, "    suspend fun %s(): GSXResponse = client.%s(\n", kotlinIdentifier(endpoint.Name), operation)
		fmt.Fprintf(buf, "        GSXRequest(method = %s, path = %s),\n", strconv.Quote(endpointMethod(endpoint, defaultMethod)), strconv.Quote(endpoint.Path))
		fmt.Fprintln(buf, "    )")
	}
	fmt.Fprintln(buf, "}")
}

func endpointMethod(endpoint endpointDeclaration, fallback string) string {
	method := strings.ToUpper(strings.TrimSpace(endpoint.Method))
	if method == "" {
		return fallback
	}
	return method
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
