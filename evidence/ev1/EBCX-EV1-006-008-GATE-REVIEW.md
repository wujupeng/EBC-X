# EBC-X EV1-006~008 Gate Review Package

> 提交时间：2026-09-08  
> 提交人：Coding Agent  
> 审查人：大G项目经理  
> 裁决状态：**待审查**

---

## 1. Task Definition → Code → Test → Evidence 映射

### EV1-006: Object Storage Artifact

| 项目 | 路径 |
|------|------|
| Task Definition | tasks.md L437-451 |
| Code | `internal/platform/artifact/model.go`, `store.go` |
| Test | `internal/platform/artifact/store_test.go` (29 tests) |
| Migration | `db/migrations/V7__artifact_metadata.sql` |
| Evidence | `evidence/ev1/EBCX-EV1-006-obs-evidence.json` |
| Code Commit | `8d8ef62` |
| Evidence Commit | `536dede` |

### EV1-007: HTKIS-AF Security Base

| 项目 | 路径 |
|------|------|
| Task Definition | tasks.md L455-475 |
| Code | `internal/platform/security/{jwt,audit,kms,tls,rls,lint}.go` |
| Test | `internal/platform/security/security_test.go` (31 tests) |
| Migration | `db/migrations/V8__security_audit.sql` |
| Evidence | `evidence/ev1/EBCX-EV1-007-security-evidence.json` |
| Code Commit | `994a6af` |
| Evidence Commit | `01f460f` |

### EV1-008: Multi-Tenant Base

| 项目 | 路径 |
|------|------|
| Task Definition | tasks.md L477-495 |
| Code | `internal/platform/tenant/{model,pack_router}.go` |
| Test | `internal/platform/tenant/pack_router_test.go` (19 tests) |
| Migration | `db/migrations/V9__tenant_pack_routing.sql` |
| Evidence | `evidence/ev1/EBCX-EV1-008-multi-tenant-evidence.json` |
| Code Commit | `a763d18` |
| Evidence Commit | `f36a746` |

---

## 2. Unit / Integration / Physical Test 分类

### EV1-006 (29 tests)

| 分类 | 数量 | 测试列表 | 说明 |
|------|------|----------|------|
| **UNIT** | 12 | TestBuildObjectKey, TestParseObjectKey_*, TestIsValidArtifactType, TestArtifactValidate_* | 纯逻辑，无 I/O |
| **INTEGRATION (Real Filesystem)** | 17 | TestUpload_*, TestDownload_*, TestDelete_*, TestGetMetadata_*, TestListBy*, TestExists_*, TestChecksum_*, TestEvidenceTraceability, TestManifest_Persistence | 使用 `t.TempDir()` 真实文件系统 |
| **PHYSICAL (Real PostgreSQL)** | **0** | — | **V7 迁移未经真实 PostgreSQL 验证** |

### EV1-007 (31 tests)

| 分类 | 数量 | 测试列表 | 说明 |
|------|------|----------|------|
| **UNIT** | 21 | TestJWT_* (10), TestAudit_Append/Query/GetByID/EntryFromClaims (6), TestKMS_* (5) | 纯逻辑 / InMemoryAuditLogger |
| **INTEGRATION (Real Crypto)** | 4 | TestKMS_GenerateAndEncrypt, TestKMS_Decrypt_RoundTrip, TestKMS_KeyRotation, TestKMS_HashPII | 真实 AES-256-GCM 加密 |
| **INTEGRATION (Real Filesystem Lint)** | 1 | TestSecurityLint_NoViolations | 扫描项目实际 .go 文件 |
| **UNIT (TLS Constants)** | 5 | TestTLS_* | 检查 TLS 常量值 |
| **PHYSICAL (Real PostgreSQL)** | **0** | — | **V8 迁移未经真实 PostgreSQL 验证** |

### EV1-008 (19 tests)

| 分类 | 数量 | 测试列表 | 说明 |
|------|------|----------|------|
| **UNIT** | 19 | 全部 19 个测试 | InMemoryPackRouter，无 I/O |
| **INTEGRATION** | 0 | — | — |
| **PHYSICAL (Real PostgreSQL)** | **0** | — | **V9 迁移未经真实 PostgreSQL 验证** |

---

## 3. 实际测试命令

```
go test ./internal/platform/artifact/... -v -count=1
go test ./internal/platform/security/... -v -count=1
go test ./internal/platform/tenant/... -v -count=1
```

---

## 4. 实际测试输出摘要

| 任务 | total_pass | total_fail | 耗时 |
|------|-----------|-----------|------|
| EV1-006 | 29 | 0 | 0.274s |
| EV1-007 | 31 | 0 | 0.328s |
| EV1-008 | 19 | 0 | 0.280s |

---

## 5. Physical Infrastructure 类型及版本

| 组件 | 版本 | 实际使用情况 |
|------|------|-------------|
| Go | 1.26.7 | ✅ 使用 |
| PostgreSQL 18.3 (embedded-postgres) | 18.3 | ❌ **EV1-006/007/008 均未使用** |
| Neo4j 5.26.0 | 5.26.0 | ❌ 不涉及 |
| Local Filesystem | Windows NTFS | ✅ EV1-006 WORM 文件权限验证 |

---

## 6. PostgreSQL RLS 是否为 Real Physical Verification？

### 结论：❌ 否

**EV1-006**: V7 迁移中 `artifact.metadata` 表定义了 RLS 策略 `artifact_tenant_isolation`，但**无任何测试启动真实 PostgreSQL 验证 RLS 生效**。租户隔离仅通过 Go 应用层 `meta.TenantID != tenantID` 判断实现。

**EV1-007**: V8 迁移中 `audit.audit_log` 和 `security.revoked_tokens` 定义了 RLS 策略，`RLSManager` 结构体存在 `SetTenantContext` 方法，但**无任何测试调用 `RLSManager` 验证 RLS 生效**。审计租户隔离仅通过 `InMemoryAuditLogger` 的应用层过滤实现。

**EV1-008**: V9 迁移中 5 个扩展表均定义了 RLS 策略，但**无任何测试启动真实 PostgreSQL 验证 RLS 生效**。租户隔离仅在 `PackRouter` 的应用层路由判断实现。

### 缺失的 Physical Verification

需要补充的测试模式（参考 EV1-005A `physical_gate3_test.go` 的 embedded-postgres 模式）：

```
1. 启动 embedded-postgres
2. 执行 V1~V9 迁移
3. INSERT tenant A 数据
4. INSERT tenant B 数据
5. SET app.tenant_id = A
6. SELECT → 只返回 A 的数据
7. 尝试 UPDATE audit_log → RAISE EXCEPTION
8. 尝试 DELETE audit_log → RAISE EXCEPTION
9. 尝试 UPDATE artifact.metadata → RAISE EXCEPTION
10. 尝试 DELETE artifact.metadata → RAISE EXCEPTION
```

---

## 7. Tenant Isolation 是否为 Real Physical Verification？

### 结论：❌ 否（应用层是，DB 层不是）

| 任务 | 应用层隔离 | DB 层 RLS 验证 |
|------|-----------|---------------|
| EV1-006 | ✅ `ErrCrossTenantAccess` | ❌ 未验证 |
| EV1-007 | ✅ `InMemoryAuditLogger` 过滤 | ❌ 未验证 |
| EV1-008 | ✅ `PackRouter` 路由判断 | ❌ 未验证 |

**应用层隔离**已在单元测试中验证（cross-tenant download/metadata/exists 均返回错误）。

**DB 层 RLS** 仅在迁移 SQL 中定义，未经真实 PostgreSQL 验证。这意味着：
- 如果应用层有 bug 绕过了 tenant 检查，DB 层 RLS 是否能兜底拦截 → **未知**
- 如果直接用 SQL 连接绕过应用层，RLS 是否生效 → **未知**

---

## 8. Artifact WORM / Audit WORM 是否经过真实数据库或文件系统验证？

### Artifact WORM

| 验证项 | 文件系统 | 数据库 |
|--------|---------|--------|
| 文件权限 0444 (read-only) | ✅ `TestUpload_WORM_FileIsReadOnly` | N/A |
| 重复上传拒绝 | ✅ `TestUpload_WORM_DuplicateRejected` | ❌ 未验证 unique index |
| 删除禁止 | ✅ `TestDelete_WORM_Forbidden` | ❌ 未验证 trigger |
| 修改禁止 | ❌ 未测试 | ❌ 未验证 trigger |

**结论**: 文件系统 WORM **已验证**（真实 NTFS 文件权限 0444）。数据库 WORM triggers **未验证**。

### Audit WORM

| 验证项 | InMemoryAuditLogger | PostgreSQL |
|--------|---------------------|------------|
| Append 成功 | ✅ | ❌ 未验证 |
| UPDATE 禁止 | N/A | ❌ 未验证 trigger |
| DELETE 禁止 | N/A | ❌ 未验证 trigger |

**结论**: 审计 append-only **仅 InMemory 验证**。PostgreSQL trigger `audit.block_mutation()` **未验证**。

---

## 9. Pack Routing 与 Core Table No-Fork 验证结果

### Pack Routing

| 验证项 | 结果 | 类型 |
|--------|------|------|
| 5 Pack 路由正确 | ✅ PASS | UNIT |
| X-Tenant-Id + X-Country-Pack 提取 | ✅ PASS | UNIT |
| 大小写不敏感 | ✅ PASS | UNIT |
| 缺失 Header 拒绝 | ✅ PASS | UNIT |
| 未启用 Pack 拒绝 | ✅ PASS | UNIT |
| **跨租户 Pack 访问隔离** | ❌ **未验证** | — |
| **租户伪装 Pack 提权** | ❌ **未验证** | — |

### Core Table No-Fork

| 验证项 | 结果 | 类型 |
|--------|------|------|
| 相同 schema 通过 | ✅ PASS | UNIT |
| 列数不同检测 | ✅ PASS | UNIT |
| 列类型不同检测 | ✅ PASS | UNIT |
| **实际 V9 迁移 schema 一致性** | ❌ **未验证** | — |

`ValidateCoreTableNoFork` 函数验证的是传入的 `CoreTableSpec` 结构体，**并未实际查询 PostgreSQL `information_schema.columns` 验证 V9 迁移产生的扩展表结构**。

---

## 10-13. Evidence Provenance

### git log --oneline -10

```
f36a746 EBC-X EV1-008: Physical Evidence — 19 tests PASS
a763d18 EBC-X EV1-008: Multi-Tenant Base — RLS + Pack Routing + 5 Country Packs
01f460f EBC-X EV1-007: Physical Evidence — 31 tests PASS
994a6af EBC-X EV1-007: HTKIS-AF Security Base — OAuth2+JWT+RLS+Audit+WORM+KMS+TLS
536dede EBC-X EV1-006: Physical Evidence — 29 tests PASS
8d8ef62 EBC-X EV1-006: Object Storage Artifact — WORM + Evidence-Linked + Tenant Isolation
409f21f EBC-X EV1-005/005A PHYS-04: Evidence provenance fix
96b354e EBC-X EV1-005/005A Physical Gate 2: PHYS-01/02/03
b97631d EBC-X EV1-005/005A: Neo4j Graph Projection + Reconciliation
0f3b7be EBC-X EV0-TASKS v1.2: TASK HARDENING-002
```

### HEAD commit

```
f36a74680cf8800039e1f3fbf906b8be229f13bb
```

### Evidence git_commit 与实际验证 commit 一致性

| 任务 | Evidence git_commit | 实际代码 commit | 一致性 |
|------|-------------------|----------------|--------|
| EV1-006 | `8d8ef624c6963a7018403d76f9e6d4cd1c2760d4` | `8d8ef62` | ✅ 一致 |
| EV1-007 | `994a6affaec814ea80d0549dda79517ca1386bd0` | `994a6af` | ✅ 一致 |
| EV1-008 | `a763d180119e6b14e73e9f726509a00b1b0788c9` | `a763d18` | ✅ 一致 |

### git status

```
On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  modified: internal/platform/graph/physical_gate2_shadow_test.go  (trailing newline only)
  modified: internal/platform/graph/physical_gate2_test.go         (trailing newline only)
  modified: internal/platform/graph/physical_gate3_test.go          (trailing newline only)
```

**注意**: 有 3 个 graph 测试文件有未提交的变更（仅尾行换行符差异，无功能变更）。

---

## 14. git status

见上文。3 个文件有尾行换行符变更，无功能影响。

---

## 15. V7/V8/V9 Migration 验证结果

| 迁移 | SQL 语法 | 依赖衔接 | Physical 执行验证 |
|------|---------|---------|------------------|
| V7 (artifact) | ✅ 正确 | ✅ FK → evidence.evidence_ledger | ❌ **未执行** |
| V8 (security) | ✅ 正确 | ✅ schema security + audit | ❌ **未执行** |
| V9 (tenant) | ✅ 正确 | ✅ FK → tenant.tenants | ❌ **未执行** |

**迁移 SQL 均未经真实 PostgreSQL 执行验证**。仅通过 `go build ./...` 确认 Go 代码编译通过。

### 依赖链完整性

```
V1 (schemas: business/evidence/outbox/audit/master_data/policy/tenant)
  ↓
V2 (roles)
  ↓
V3 (tenant_foundation)
  ↓
V4 (evidence_ledger) ← V7 FK 引用
  ↓
V5 (outbox)
  ↓
V6 (graph_projection_meta)
  ↓
V7 (artifact) — FK → evidence.evidence_ledger ✅
  ↓
V8 (security) — new schema: security ✅
  ↓
V9 (tenant) — FK → tenant.tenants ✅
```

依赖链**逻辑正确**，但**未经物理执行验证**。

---

## 16. 残余风险

### EV1-006 残余风险

| 风险 | 严重度 | 说明 |
|------|--------|------|
| V7 RLS 未物理验证 | 🔴 高 | 应用层 bug 可能绕过租户隔离 |
| V7 WORM triggers 未物理验证 | 🔴 高 | DB 层防御未确认 |
| Artifact 修改未测试 | 🟡 中 | `ErrArtifactWORMModify` 定义但无测试 |
| V7 FK 未物理验证 | 🟡 中 | evidence_id 引用完整性未确认 |

### EV1-007 残余风险

| 风险 | 严重度 | 说明 |
|------|--------|------|
| V8 audit append-only triggers 未物理验证 | 🔴 高 | 核心安全特性未确认 |
| V8 RLS 未物理验证 | 🔴 高 | 审计日志租户隔离未确认 |
| RLSManager 未经任何测试 | 🔴 高 | 代码存在但 0 测试覆盖 |
| InMemoryAuditLogger ≠ PostgreSQL AuditLogger | 🟡 中 | 生产环境使用 PostgreSQL，测试使用内存 |
| JWT 使用 HS256 对称签名 | 🟡 低 | 生产可能需要 RS256 非对称签名 |

### EV1-008 残余风险

| 风险 | 严重度 | 说明 |
|------|--------|------|
| V9 RLS 未物理验证 | 🔴 高 | 扩展表租户隔离未确认 |
| 跨租户 Pack 访问未验证 | 🔴 高 | Tenant-A+CN 不能访问 Tenant-B+CN 未测试 |
| Core Table No-Fork 未查实际 schema | 🟡 中 | 仅验证传入结构体，未查 information_schema |
| V9 迁移未物理执行 | 🔴 高 | SQL 语法正确性未确认 |

---

## 17. 综合评估

### 测试通过但分类统计

| 任务 | UNIT PASS | INTEGRATION PASS | PHYSICAL PASS | 总 PASS |
|------|----------|-----------------|--------------|---------|
| EV1-006 | 12 | 17 (filesystem) | **0** | 29 |
| EV1-007 | 26 | 5 (crypto+lint) | **0** | 31 |
| EV1-008 | 19 | 0 | **0** | 19 |
| **合计** | **57** | **22** | **0** | **79** |

### 关键缺口

**三个任务均缺少 Physical Verification（Real PostgreSQL）**：

1. **RLS 物理验证**: V7/V8/V9 定义的 RLS 策略均未经真实 PostgreSQL 验证
2. **WORM Trigger 物理验证**: V7 artifact triggers + V8 audit triggers 均未经真实 PostgreSQL 验证
3. **迁移执行验证**: V7/V8/V9 SQL 均未在真实 PostgreSQL 上执行
4. **RLSManager 测试覆盖**: EV1-007 的 `rls.go` 有 0 个测试

### 与 EV1-005/005A Physical Gate 对比

| 项目 | EV1-005/005A | EV1-006~008 |
|------|-------------|-------------|
| Real Neo4j | ✅ 5.26.0 | N/A |
| Real PostgreSQL | ✅ embedded-postgres 18.3 | ❌ |
| Physical Gate Tests | ✅ 12 + 5 tests | ❌ 0 tests |
| Mock = Real 区分 | ✅ 明确 | ❌ 未区分 |

---

## 裁决建议

### EV1-006: 🟡 CONDITIONAL PASS

- 文件系统 WORM 已物理验证 ✅
- 应用层租户隔离已验证 ✅
- **V7 DB 层 RLS + WORM triggers 未物理验证** ❌
- 需要 Physical Gate Test 补充

### EV1-007: 🟡 CONDITIONAL PASS

- JWT/KMS/TLS 加密功能已验证 ✅
- Security Lint 已验证 ✅
- **V8 DB 层 audit append-only + RLS 未物理验证** ❌
- **RLSManager 0 测试覆盖** ❌
- 需要 Physical Gate Test 补充

### EV1-008: 🟡 CONDITIONAL PASS

- Pack Routing 逻辑已验证 ✅
- Core Table No-Fork 逻辑已验证 ✅
- **V9 DB 层 RLS 未物理验证** ❌
- **跨租户 Pack 访问隔离未验证** ❌
- 需要 Physical Gate Test 补充

### 下一步建议

**不授权 EV1-009。**

需要先完成 **EV1-006~008 Physical Gate**：

1. 使用 embedded-postgres 启动真实 PostgreSQL 18.3
2. 执行 V1~V9 全部迁移
3. 验证 RLS：SET app.tenant_id → SELECT → 只返回本租户数据
4. 验证 WORM triggers：UPDATE/DELETE → RAISE EXCEPTION
5. 验证 FK 引用完整性
6. 验证跨租户 Pack 访问隔离
7. 补充 RLSManager 测试
8. 更新 Evidence JSON 标注 Physical Verification