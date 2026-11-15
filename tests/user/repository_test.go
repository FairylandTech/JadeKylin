/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 11:43:44 America/Los_Angeles
 *****************************************************************************/

package user_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
	"jkylin.com/kylin/v2/internal/pkg/valueobject"
)

func verifyRoleAndPermissionRepositories(t *testing.T, database *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	roleRepository := role.NewRepository(database)
	permissionRepository := permission.NewRepository(database)

	admin, err := roleRepository.GetByCode(ctx, "admin")
	if err != nil || admin.Permissions == nil || len(*admin.Permissions) == 0 {
		t.Fatalf("load seeded role: %+v, %v", admin, err)
	}
	for _, code := range *admin.Permissions {
		if _, err := permissionRepository.GetByCode(ctx, code); err != nil {
			t.Fatalf("seeded permission %q: %v", code, err)
		}
	}

	createdPermission := &permission.Permission{Code: "test.read", Name: "Test Read", Category: "Test Category"}
	if err := permissionRepository.Create(ctx, createdPermission); err != nil || createdPermission.ID <= 0 || createdPermission.Enabled != "Y" {
		t.Fatalf("create permission: %+v, %v", createdPermission, err)
	}
	if err := permissionRepository.Create(ctx, &permission.Permission{Code: "test.duplicate", Name: createdPermission.Name, Category: createdPermission.Category}); !errors.Is(err, persistence.ErrDuplicate) {
		t.Fatalf("duplicate permission name: %v", err)
	}
	updatedPermission, err := permissionRepository.Update(ctx, createdPermission.ID, map[string]any{"description": "Updated"})
	if err != nil || updatedPermission.Description != "Updated" || updatedPermission.Code != createdPermission.Code {
		t.Fatalf("update permission: %+v, %v", updatedPermission, err)
	}
	allPermissions, err := permissionRepository.List(ctx)
	if err != nil || !slices.ContainsFunc(allPermissions, func(candidate permission.Permission) bool {
		return candidate.ID == createdPermission.ID
	}) ||
		len(allPermissions) <= len(*admin.Permissions) {
		t.Fatalf("list permissions: %+v, %v", allPermissions, err)
	}
	if existing, err := permissionRepository.ExistingCodes(ctx, []string{createdPermission.Code, "no.such.permission"}); err != nil || !slices.Equal(existing, []string{createdPermission.Code}) {
		t.Fatalf("existing permission codes: %v, %v", existing, err)
	}

	createdRole := &role.Role{Code: "test_role", Name: "Test Role"}
	if err := roleRepository.Create(ctx, createdRole); err != nil || createdRole.ID <= 0 || createdRole.Permissions == nil || len(*createdRole.Permissions) != 0 {
		t.Fatalf("create role with default permissions: %+v, %v", createdRole, err)
	}
	if err := roleRepository.Create(ctx, &role.Role{Code: createdRole.Code, Name: "Duplicate Role"}); !errors.Is(err, persistence.ErrDuplicate) {
		t.Fatalf("duplicate role code: %v", err)
	}
	wantPermissions := valueobject.PermissionList{createdPermission.Code, `with,"quotes"`}
	updatedRole, err := roleRepository.Update(ctx, createdRole.ID, map[string]any{"permissions": &wantPermissions})
	if err != nil || updatedRole.Permissions == nil || !slices.Equal(*updatedRole.Permissions, wantPermissions) {
		t.Fatalf("update role: %+v, %v", updatedRole, err)
	}
	fetchedRole, err := roleRepository.Get(ctx, createdRole.ID)
	if err != nil || fetchedRole.Permissions == nil || !slices.Equal(*fetchedRole.Permissions, wantPermissions) {
		t.Fatalf("get role: %+v, %v", fetchedRole, err)
	}
	rolePage, total, err := roleRepository.List(ctx, "TEST_ROLE", 1, 1)
	if err != nil || total != 1 || len(rolePage) != 0 {
		t.Fatalf("paginate roles: %+v, %d, %v", rolePage, total, err)
	}
	if err := roleRepository.Delete(ctx, createdRole.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := roleRepository.GetByCode(ctx, createdRole.Code); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("get disabled role: %v", err)
	}
	if _, err := roleRepository.Update(ctx, createdRole.ID, map[string]any{"name": "Disabled"}); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("update disabled role: %v", err)
	}
	if err := permissionRepository.Delete(ctx, createdPermission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := permissionRepository.Get(ctx, createdPermission.ID); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("get disabled permission: %v", err)
	}
	if err := permissionRepository.Delete(ctx, createdPermission.ID); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("delete disabled permission: %v", err)
	}
}
