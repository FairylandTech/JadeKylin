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
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/repository"
)

func verifyRoleAndPermissionRepositories(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	roles := repository.NewRoleRepository(db)
	permissions := repository.NewPermissionRepository(db)

	admin, err := roles.GetByCode(ctx, "admin")
	if err != nil || admin.Permissions == nil || len(*admin.Permissions) == 0 {
		t.Fatalf("load seeded role: %+v, %v", admin, err)
	}
	for _, code := range *admin.Permissions {
		if _, err := permissions.GetByCode(ctx, code); err != nil {
			t.Fatalf("seeded permission %q: %v", code, err)
		}
	}

	permission := &model.Permission{Code: "test.read", Name: "Test Read", Category: "Test Category"}
	if err := permissions.Create(ctx, permission); err != nil || permission.ID <= 0 || permission.Enabled != "Y" {
		t.Fatalf("create permission: %+v, %v", permission, err)
	}
	if err := permissions.Create(ctx, &model.Permission{Code: "test.duplicate", Name: permission.Name, Category: permission.Category}); !errors.Is(err, support.ErrDuplicate) {
		t.Fatalf("duplicate permission name: %v", err)
	}
	updatedPermission, err := permissions.Update(ctx, permission.ID, map[string]any{"description": "Updated"})
	if err != nil || updatedPermission.Description != "Updated" || updatedPermission.Code != permission.Code {
		t.Fatalf("update permission: %+v, %v", updatedPermission, err)
	}
	allPermissions, err := permissions.List(ctx)
	if err != nil || !slices.ContainsFunc(allPermissions, func(p model.Permission) bool { return p.ID == permission.ID }) ||
		len(allPermissions) <= len(*admin.Permissions) {
		t.Fatalf("list permissions: %+v, %v", allPermissions, err)
	}
	if existing, err := permissions.ExistingCodes(ctx, []string{permission.Code, "no.such.permission"}); err != nil || !slices.Equal(existing, []string{permission.Code}) {
		t.Fatalf("existing permission codes: %v, %v", existing, err)
	}

	role := &model.Role{Code: "test_role", Name: "Test Role"}
	if err := roles.Create(ctx, role); err != nil || role.ID <= 0 || role.Permissions == nil || len(*role.Permissions) != 0 {
		t.Fatalf("create role with default permissions: %+v, %v", role, err)
	}
	if err := roles.Create(ctx, &model.Role{Code: role.Code, Name: "Duplicate Role"}); !errors.Is(err, support.ErrDuplicate) {
		t.Fatalf("duplicate role code: %v", err)
	}
	wantPermissions := valueobject.PermissionList{permission.Code, `with,"quotes"`}
	updatedRole, err := roles.Update(ctx, role.ID, map[string]any{"permissions": &wantPermissions})
	if err != nil || updatedRole.Permissions == nil || !slices.Equal(*updatedRole.Permissions, wantPermissions) {
		t.Fatalf("update role: %+v, %v", updatedRole, err)
	}
	fetchedRole, err := roles.Get(ctx, role.ID)
	if err != nil || fetchedRole.Permissions == nil || !slices.Equal(*fetchedRole.Permissions, wantPermissions) {
		t.Fatalf("get role: %+v, %v", fetchedRole, err)
	}
	rolePage, total, err := roles.List(ctx, "TEST_ROLE", 1, 1)
	if err != nil || total != 1 || len(rolePage) != 0 {
		t.Fatalf("paginate roles: %+v, %d, %v", rolePage, total, err)
	}
	if err := roles.Delete(ctx, role.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := roles.GetByCode(ctx, role.Code); !errors.Is(err, support.ErrNotFound) {
		t.Fatalf("get disabled role: %v", err)
	}
	if _, err := roles.Update(ctx, role.ID, map[string]any{"name": "Disabled"}); !errors.Is(err, support.ErrNotFound) {
		t.Fatalf("update disabled role: %v", err)
	}
	if err := permissions.Delete(ctx, permission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := permissions.Get(ctx, permission.ID); !errors.Is(err, support.ErrNotFound) {
		t.Fatalf("get disabled permission: %v", err)
	}
	if err := permissions.Delete(ctx, permission.ID); !errors.Is(err, support.ErrNotFound) {
		t.Fatalf("delete disabled permission: %v", err)
	}
}
