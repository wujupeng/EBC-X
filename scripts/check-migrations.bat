@echo off
REM EBC-X Migration CI Check — verify no manual schema changes in production
REM Usage: scripts\check-migrations.bat

echo === EBC-X Migration CI Check ===

REM Verify all migration files are versioned (V{n}__*.sql pattern)
setlocal enabledelayedexpansion
set count=0
for %%f in (db\migrations\V*.sql) do (
    set /a count+=1
    echo   [OK] %%f
)
echo Total migration files: !count!

REM Verify no unversioned SQL files
set unversioned=0
for %%f in (db\migrations\*.sql) do (
    echo %%f | findstr /r "^V[0-9]" >nul
    if errorlevel 1 (
        echo   [WARN] Unversioned: %%f
        set /a unversioned+=1
    )
)
if !unversioned! gtr 0 (
    echo ❌ !unversioned! unversioned SQL files found
    exit /b 1
)

REM Verify Runtime Role has no UPDATE/DELETE/TRUNCATE on evidence (TASK-H02)
findstr /i "REVOKE.*UPDATE.*DELETE.*TRUNCATE.*evidence" db\migrations\V2__create_roles.sql >nul
if errorlevel 1 (
    echo ❌ TASK-H02 violation: Runtime Role REVOKE not found
    exit /b 1
) else (
    echo   [OK] TASK-H02: Runtime Role REVOKE UPDATE/DELETE/TRUNCATE on evidence
)

echo ✅ Migration CI check passed
endlocal