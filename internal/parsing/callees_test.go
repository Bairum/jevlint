package parsing

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestExtractRecordsQualifiedCallRefs(t *testing.T) {
	t.Parallel()

	units := extractGoFunctions(t, "users.go", `package sample

func buildUsers() {
	users := loadUsers()
	users = loadUsers()
	accounts := loadAccounts()
	store.Users()
	user.Save()
	fmt.Println(users, accounts)
}

func loadUsers() {}
func loadAccounts() {}
`)
	build := functionNamed(t, units, "buildUsers")
	got := spellings(build.CallRefs)
	want := []string{"loadUsers", "loadAccounts", "store.Users", "user.Save", "fmt.Println"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("call refs = %#v, want %#v", got, want)
	}
	if build.Callees != nil {
		t.Fatalf("extracted callees = %#v, want omitted", build.Callees)
	}
}

func TestExtractOmitsNestedFunctionCallsFromOuterUnit(t *testing.T) {
	t.Parallel()

	units := extractGoFunctions(t, "nested.go", `package sample

func outer() {
	inner()
}

func inner() {
	secret()
}

func secret() {}
`)
	outer := functionNamed(t, units, "outer")
	if got := spellings(outer.CallRefs); !reflect.DeepEqual(got, []string{"inner"}) {
		t.Fatalf("outer call refs = %#v", got)
	}
}

func TestResolveCalleesConservativeMatches(t *testing.T) {
	t.Parallel()

	t.Run("same-file unique callee", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "users.go", `package sample
func buildUsers() { loadUsers() }
func loadUsers() {}
`)
		resolveExtracted(units)
		got := calleeNames(functionNamed(t, units, "buildUsers").Resolved)
		if !reflect.DeepEqual(got, []string{"loadUsers"}) {
			t.Fatalf("resolved = %#v", got)
		}
	})

	t.Run("multiple direct callees keep source order", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "users.go", `package sample
func buildUsers() {
	loadUsers()
	loadAccounts()
	combine()
}
func loadUsers() {}
func loadAccounts() {}
func combine() {}
`)
		resolveExtracted(units)
		got := calleeNames(functionNamed(t, units, "buildUsers").Resolved)
		if !reflect.DeepEqual(got, []string{"loadUsers", "loadAccounts", "combine"}) {
			t.Fatalf("resolved = %#v", got)
		}
	})

	t.Run("duplicate calls collapse to one callee", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "users.go", `package sample
func buildUsers() {
	loadUsers()
	loadUsers()
}
func loadUsers() {}
`)
		resolveExtracted(units)
		got := calleeNames(functionNamed(t, units, "buildUsers").Resolved)
		if !reflect.DeepEqual(got, []string{"loadUsers"}) {
			t.Fatalf("resolved = %#v", got)
		}
	})

	t.Run("unique cross-file callee", func(t *testing.T) {
		t.Parallel()
		users := extractGoFunctions(t, "users.go", `package sample
func buildUsers() { loadAccounts() }
`)
		accounts := extractGoFunctions(t, "accounts.go", `package sample
func loadAccounts() {}
`)
		all := append(functionPointers(users), functionPointers(accounts)...)
		ResolveCallees(all)
		got := calleeNames(functionNamed(t, users, "buildUsers").Resolved)
		if !reflect.DeepEqual(got, []string{"loadAccounts"}) {
			t.Fatalf("resolved = %#v", got)
		}
		if functionNamed(t, users, "buildUsers").Resolved[0].Path != "accounts.go" {
			t.Fatalf("callee path = %#v", functionNamed(t, users, "buildUsers").Resolved[0])
		}
	})

	t.Run("ambiguous project match omitted", func(t *testing.T) {
		t.Parallel()
		caller := extractGoFunctions(t, "run.go", `package sample
func run() { helper() }
`)
		left := extractGoFunctions(t, "left.go", `package sample
func helper() {}
`)
		right := extractGoFunctions(t, "right.go", `package sample
func helper() {}
`)
		ResolveCallees(append(
			append(functionPointers(caller), functionPointers(left)...),
			functionPointers(right)...,
		))
		if got := functionNamed(t, caller, "run").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("ambiguous Save methods omitted", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "models.go", `package sample
func (User) Save() {}
func (Order) Save() {}
func (Cache) Save() {}
func persist() {
	user.Save()
	order.Save()
	cache.Save()
}
`)
		resolveExtracted(units)
		if got := functionNamed(t, units, "persist").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
		if got := spellings(functionNamed(t, units, "persist").CallRefs); !reflect.DeepEqual(
			got,
			[]string{"user.Save", "order.Save", "cache.Save"},
		) {
			t.Fatalf("call refs = %#v", got)
		}
	})

	t.Run("external call omitted", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "users.go", `package sample
func buildUsers() { fmt.Println("users") }
`)
		resolveExtracted(units)
		if got := functionNamed(t, units, "buildUsers").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("qualified external call omitted despite same-named project function", func(t *testing.T) {
		t.Parallel()
		caller := extractGoFunctions(t, "caller.go", `package sample
func run() { foo.Parse("users") }
`)
		parse := extractGoFunctions(t, "parse.go", `package sample
func Parse() {}
`)
		ResolveCallees(append(functionPointers(caller), functionPointers(parse)...))
		if got := functionNamed(t, caller, "run").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("qualified same-file call omitted", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "same.go", `package sample
func run() { foo.Parse("users") }
func Parse() {}
`)
		resolveExtracted(units)
		if got := functionNamed(t, units, "run").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("qualified call omitted even with same-file candidate", func(t *testing.T) {
		t.Parallel()
		local := extractGoFunctions(t, "local.go", `package sample
func run() { foo.Parse("users") }
func Parse() {}
`)
		remote := extractGoFunctions(t, "remote.go", `package sample
func Parse() {}
`)
		ResolveCallees(append(functionPointers(local), functionPointers(remote)...))
		if got := functionNamed(t, local, "run").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("qualified method omitted despite unique method", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "models.go", `package sample
func (User) Save() {}
func persist() { user.Save() }
`)
		resolveExtracted(units)
		if got := functionNamed(t, units, "persist").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("unqualified call still resolves beside qualified call", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "mixed.go", `package sample
func run() {
	helper()
	log.Println(helper())
}
func helper() {}
`)
		resolveExtracted(units)
		got := calleeNames(functionNamed(t, units, "run").Resolved)
		if !reflect.DeepEqual(got, []string{"helper"}) {
			t.Fatalf("resolved = %#v, want [helper]", got)
		}
	})

	t.Run("self-recursive call omitted", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "walk.go", `package sample
func walk() { walk() }
`)
		resolveExtracted(units)
		if got := functionNamed(t, units, "walk").Resolved; got != nil {
			t.Fatalf("resolved = %#v, want omitted", got)
		}
	})

	t.Run("callee callees are not expanded", func(t *testing.T) {
		t.Parallel()
		units := extractGoFunctions(t, "chain.go", `package sample
func a() { b() }
func b() { c() }
func c() {}
`)
		resolveExtracted(units)
		got := calleeNames(functionNamed(t, units, "a").Resolved)
		if !reflect.DeepEqual(got, []string{"b"}) {
			t.Fatalf("resolved = %#v", got)
		}
	})
}

func TestExpandCalleesEnforcesBudgets(t *testing.T) {
	t.Parallel()

	resolved := make([]CalleeContext, 0, maxDirectCallees+1)
	for index := range maxDirectCallees + 1 {
		resolved = append(resolved, CalleeContext{
			Name:   "fn",
			Path:   "f.go",
			Source: strings.Repeat("x", 8),
		})
		resolved[index].Name = "fn" + strings.Repeat("A", index+1)
	}
	got := ExpandCallees(resolved)
	if len(got) != maxDirectCallees {
		t.Fatalf("count = %d, want %d", len(got), maxDirectCallees)
	}

	large := CalleeContext{Name: "large", Source: strings.Repeat("L", maxCalleeSourceBytes+1)}
	small := CalleeContext{Name: "small", Source: "func small() {}"}
	expanded := ExpandCallees([]CalleeContext{large, small})
	if len(expanded) != 1 || expanded[0].Name != "small" {
		t.Fatalf("over-budget omit = %#v", expanded)
	}
	if strings.Contains(expanded[0].Source, "L") {
		t.Fatal("over-budget callee was truncated instead of omitted")
	}

	first := CalleeContext{Name: "first", Source: strings.Repeat("a", 100)}
	second := CalleeContext{Name: "second", Source: strings.Repeat("b", 100)}
	if got := ExpandCallees([]CalleeContext{first, second}); !reflect.DeepEqual(
		calleeNames(got),
		[]string{"first", "second"},
	) {
		t.Fatalf("order = %#v", got)
	}
}

func TestExtractedUnitJSONOmitsCalleesByDefault(t *testing.T) {
	t.Parallel()

	units := extractGoFunctions(t, "users.go", `package sample
func buildUsers() { loadUsers() }
func loadUsers() {}
`)
	raw, err := json.Marshal(functionNamed(t, units, "buildUsers"))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(raw), `"callees"`) {
		t.Fatalf("extracted JSON included callees: %s", raw)
	}
	if strings.Contains(string(raw), `"callRefs"`) {
		t.Fatalf("extracted JSON included call refs: %s", raw)
	}

	resolveExtracted(units)
	enriched := WithCalleeContext(functionNamed(t, units, "buildUsers"))
	raw, err = json.Marshal(enriched)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	callees, ok := payload["callees"].([]any)
	if !ok || len(callees) != 1 {
		t.Fatalf("enriched JSON callees = %#v", payload["callees"])
	}
}

func extractGoFunctions(t *testing.T, path string, source string) []CodeUnit {
	t.Helper()
	units, err := testExtractor(t, "go").Extract(path, []byte(source))
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	return units
}

func resolveExtracted(units []CodeUnit) {
	ResolveCallees(functionPointers(units))
}

func functionPointers(units []CodeUnit) []*CodeUnit {
	functions := make([]*CodeUnit, 0, len(units))
	for index := range units {
		if units[index].Kind == CodeKindFunction {
			functions = append(functions, &units[index])
		}
	}
	return functions
}

func functionNamed(t *testing.T, units []CodeUnit, name string) CodeUnit {
	t.Helper()
	for _, unit := range units {
		if unit.Kind == CodeKindFunction && unit.Name == name {
			return unit
		}
	}
	t.Fatalf("missing function %q", name)
	return CodeUnit{}
}

func spellings(refs []CallRef) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.Spelling())
	}
	return names
}

func calleeNames(callees []CalleeContext) []string {
	names := make([]string, 0, len(callees))
	for _, callee := range callees {
		names = append(names, callee.Name)
	}
	return names
}
