package models

import "testing"

func TestTenantRole_DefaultPermissions_Expected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		role       TenantRole
		permission TenantPermission
		want       bool
	}{
		{name: "viewer inventory", role: TenantRoleViewer, permission: TenantPermissionInventoryRead, want: true},
		{name: "viewer migration denied", role: TenantRoleViewer, permission: TenantPermissionMigrationManage, want: false},
		{name: "operator migration", role: TenantRoleOperator, permission: TenantPermissionMigrationManage, want: true},
		{name: "operator tenant manage denied", role: TenantRoleOperator, permission: TenantPermissionTenantManage, want: false},
		{name: "admin tenant manage", role: TenantRoleAdmin, permission: TenantPermissionTenantManage, want: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			account := ServiceAccount{Role: test.role}
			if got := account.Allows(test.permission); got != test.want {
				t.Fatalf("ServiceAccount{Role:%q}.Allows(%q) = %t, want %t", test.role, test.permission, got, test.want)
			}
		})
	}
}

func TestServiceAccount_EffectivePermissions_ExplicitPermissionsOverrideRole_Expected(t *testing.T) {
	t.Parallel()

	account := ServiceAccount{
		Role: TenantRoleAdmin,
		Permissions: []TenantPermission{
			TenantPermissionInventoryRead,
			TenantPermissionTenantRead,
		},
	}

	if !account.Allows(TenantPermissionInventoryRead) {
		t.Fatal("Allows(inventory.read) = false, want true")
	}
	if account.Allows(TenantPermissionMigrationManage) {
		t.Fatal("Allows(migration.manage) = true, want false because explicit permissions override role defaults")
	}
	if account.Allows(TenantPermissionTenantManage) {
		t.Fatal("Allows(tenant.manage) = true, want false because explicit permissions override role defaults")
	}
}

func TestTenantPermission_ValidValues_Expected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		permission TenantPermission
		want       bool
	}{
		{name: "inventory", permission: TenantPermissionInventoryRead, want: true},
		{name: "reports", permission: TenantPermissionReportsRead, want: true},
		{name: "lifecycle", permission: TenantPermissionLifecycleRead, want: true},
		{name: "migration", permission: TenantPermissionMigrationManage, want: true},
		{name: "tenant read", permission: TenantPermissionTenantRead, want: true},
		{name: "tenant manage", permission: TenantPermissionTenantManage, want: true},
		{name: "unknown", permission: TenantPermission("unknown"), want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.permission.Valid(); got != test.want {
				t.Fatalf("TenantPermission(%q).Valid() = %t, want %t", test.permission, got, test.want)
			}
		})
	}
}
