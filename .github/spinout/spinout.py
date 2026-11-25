#!/usr/bin/env python3
"""Spin a finished rebuild out of revived into its own repo.

Each run looks at the tool folders on main, oldest first, and spins out at
most one: the first that is not already its own repo and passes the gate.
The new repo keeps the folder's commit history, gets the view-only license,
a release workflow, and a v0.1.0 pre-release that builds its binaries.

The gate:
  - a real entry point (a binary, a console script or a main package)
  - it builds, and its tests pass, with at least MIN_TESTS tests
  - `<tool> --help` runs and exits 0
  - the README credits the original and shows the tool being run
  - no repo with the same name, and no repo already rebuilding the same original
"""
from __future__ import annotations

import argparse
import base64
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import tomllib
from dataclasses import dataclass, field
from pathlib import Path

MIN_TESTS = 5
MIN_README_CHARS = 400
VERSION = "v0.1.0"
HERE = Path(__file__).resolve().parent
TEMPLATES = HERE / "templates"
LICENSE_FILE = HERE / "LICENSE"
NOTICE_FILE = HERE / "readme-notice.md"
SKIP_DIRS = {".github", ".git", "docs", "scripts"}
GITHUB_LINK = re.compile(r"github\.com/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)")


class GateFail(Exception):
    pass


@dataclass
class Project:
    tool: str
    kind: str
    bin: str
    entry: str = ""
    description: str = ""
    original: str = ""
    test_cmd: str = ""
    extra: dict = field(default_factory=dict)


# ---------------------------------------------------------------- helpers

def log(msg: str) -> None:
    print(msg, flush=True)


def run(cmd, cwd=None, env=None, check=True, timeout=1800, quiet=False):
    shown = " ".join(str(c) for c in cmd)
    if not quiet:
        log(f"$ {shown}")
    proc = subprocess.run(
        [str(c) for c in cmd], cwd=cwd, env=env, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout,
    )
    if check and proc.returncode != 0:
        tail = "\n".join(proc.stdout.splitlines()[-40:])
        raise GateFail(f"`{shown}` exited {proc.returncode}\n{tail}")
    return proc


def gh_api(method: str, path: str, body: dict | None = None, check=True):
    cmd = ["gh", "api", "-X", method, path, "-H", "Accept: application/vnd.github+json"]
    if body is not None:
        cmd += ["--input", "-"]
    proc = subprocess.run(cmd, input=json.dumps(body) if body is not None else None,
                          text=True, capture_output=True)
    if check and proc.returncode != 0:
        raise RuntimeError(f"gh api {method} {path} failed: {proc.stderr.strip() or proc.stdout.strip()}")
    try:
        return proc.returncode, json.loads(proc.stdout) if proc.stdout.strip() else None
    except json.JSONDecodeError:
        return proc.returncode, None


def existing_repos(owner: str) -> list[dict]:
    """Every repo the owner has, private ones too when the token can see them."""
    for path in (f"/user/repos?affiliation=owner&per_page=100",
                 f"/users/{owner}/repos?type=owner&per_page=100"):
        proc = subprocess.run(["gh", "api", "--paginate", path], text=True, capture_output=True)
        if proc.returncode == 0:
            # --paginate glues arrays together as ][
            text = re.sub(r"\]\s*\[", ",", proc.stdout.strip() or "[]")
            repos = json.loads(text)
            if path.startswith("/user/"):
                repos = [r for r in repos if r.get("owner", {}).get("login", "").lower() == owner.lower()]
            return repos
    raise RuntimeError("could not list the owner's repos")


def norm(name: str) -> str:
    return re.sub(r"[^a-z0-9]", "", name.lower())


def summary(line: str) -> None:
    path = os.environ.get("GITHUB_STEP_SUMMARY")
    if path:
        with open(path, "a", encoding="utf-8") as fh:
            fh.write(line + "\n")


# ---------------------------------------------------------------- discovery

def tool_folders(root: Path) -> list[str]:
    tools = []
    for d in sorted(root.iterdir()):
        if not d.is_dir() or d.name in SKIP_DIRS or d.name.startswith("."):
            continue
        if not (d / "README.md").exists():
            continue
        tools.append(d.name)

    def added_at(name: str) -> int:
        out = run(["git", "log", "--format=%ct", "--", name], cwd=root, check=False, quiet=True).stdout.split()
        return int(out[-1]) if out else 1 << 62

    return sorted(tools, key=added_at)


def find_original(readme: str, owner: str) -> str:
    for line in readme.splitlines()[:40]:
        for m in GITHUB_LINK.finditer(line):
            who, repo = m.group(1), m.group(2).removesuffix(".git").rstrip(".")
            if who.lower() != owner.lower():
                return f"{who}/{repo}"
    return ""


def detect(tool: str, path: Path, owner: str) -> Project:
    readme = (path / "README.md").read_text(encoding="utf-8")
    original = find_original(readme, owner)

    if (path / "Cargo.toml").exists():
        cargo = tomllib.loads((path / "Cargo.toml").read_text(encoding="utf-8"))
        pkg = cargo.get("package", {})
        bins = cargo.get("bin", [])
        if bins:
            name = bins[0].get("name") or pkg.get("name")
        elif (path / "src" / "main.rs").exists():
            name = pkg.get("name")
        else:
            raise GateFail("rust crate has no binary (no src/main.rs or [[bin]])")
        return Project(tool, "rust", name, description=pkg.get("description", ""), original=original,
                       extra={"locked": (path / "Cargo.lock").exists()})

    if (path / "go.mod").exists():
        mod = re.search(r"^module\s+(\S+)", (path / "go.mod").read_text(encoding="utf-8"), re.M)
        main_pkg = ""
        if any(re.search(r"^package main\b", f.read_text(encoding="utf-8"), re.M) for f in path.glob("*.go")):
            main_pkg = "."
        else:
            for d in sorted((path / "cmd").glob("*")) if (path / "cmd").is_dir() else []:
                if any(re.search(r"^package main\b", f.read_text(encoding="utf-8"), re.M) for f in d.glob("*.go")):
                    main_pkg = f"./cmd/{d.name}"
                    break
        if not main_pkg:
            raise GateFail("go module has no main package")
        return Project(tool, "go", tool, original=original,
                       extra={"module": mod.group(1) if mod else "", "main": main_pkg})

    if (path / "pyproject.toml").exists():
        py = tomllib.loads((path / "pyproject.toml").read_text(encoding="utf-8"))
        proj = py.get("project", {})
        scripts = proj.get("scripts", {})
        if not scripts:
            raise GateFail("python project has no [project.scripts] entry point")
        name, entry = next(iter(scripts.items()))
        uses_pytest = any("pytest" in f.read_text(encoding="utf-8", errors="ignore")
                          for f in path.rglob("test*.py"))
        if uses_pytest:
            test_cmd = "python -m pytest -q"
        elif (path / "tests" / "__init__.py").exists():
            test_cmd = "python -m unittest discover -s tests -t . -v"
        else:
            test_cmd = "python -m unittest discover -s tests -v"
        return Project(tool, "python", name, entry=entry, description=proj.get("description", ""),
                       original=original, test_cmd=test_cmd, extra={"pkg_name": proj.get("name", "")})

    if (path / "package.json").exists():
        pkg = json.loads((path / "package.json").read_text(encoding="utf-8"))
        b = pkg.get("bin")
        if isinstance(b, str):
            name, entry = pkg.get("name", tool).split("/")[-1], b
        elif isinstance(b, dict) and b:
            name, entry = next(iter(b.items()))
        else:
            raise GateFail("node package has no bin entry point")
        if "test" not in pkg.get("scripts", {}):
            raise GateFail("node package has no test script")
        return Project(tool, "node", name, entry=entry, description=pkg.get("description", ""),
                       original=original, extra={"lock": (path / "package-lock.json").exists()})

    raise GateFail("no Cargo.toml, go.mod, pyproject.toml or package.json")


# ---------------------------------------------------------------- the gate

def count(pattern: str, text: str) -> int:
    return sum(int(n) for n in re.findall(pattern, text, re.M))


def check_readme(p: Project, path: Path) -> None:
    readme = (path / "README.md").read_text(encoding="utf-8")
    if len(readme) < MIN_README_CHARS:
        raise GateFail(f"README is under {MIN_README_CHARS} characters")
    if not p.original:
        raise GateFail("README doesn't link the original project")
    blocks = re.findall(r"```[^\n]*\n(.*?)```", readme, re.S)
    # indented code blocks count too
    blocks.append("\n".join(l.strip() for l in readme.splitlines() if l.startswith(("    ", "\t"))))
    if not any(re.search(rf"(^|[\s$>/]){re.escape(p.bin)}\b", b, re.M) for b in blocks):
        raise GateFail(f"README has no code block that runs `{p.bin}`")


def gate(p: Project, path: Path, scratch: Path) -> int:
    """Build, test and smoke-run the tool. Returns the number of tests that passed."""
    env = dict(os.environ)
    if p.kind == "rust":
        locked = ["--locked"] if p.extra["locked"] else []
        run(["cargo", "build", "--release", *locked], cwd=path)
        out = run(["cargo", "test", "--release", *locked], cwd=path).stdout
        tests = count(r"test result: ok\. (\d+) passed", out)
        binary = path / "target" / "release" / p.bin
    elif p.kind == "go":
        binary = scratch / "bin" / p.bin
        run(["go", "vet", "./..."], cwd=path)
        run(["go", "build", "-o", binary, p.extra["main"]], cwd=path)
        out = run(["go", "test", "-count=1", "-v", "./..."], cwd=path).stdout
        tests = len(re.findall(r"^\s*--- PASS", out, re.M))
    elif p.kind == "python":
        venv = scratch / "venv"
        run([sys.executable, "-m", "venv", venv])
        py = venv / "bin" / "python"
        env["PATH"] = f"{venv / 'bin'}{os.pathsep}{env['PATH']}"
        run([py, "-m", "pip", "install", "-q", "--upgrade", "pip"], env=env)
        if "pytest" in p.test_cmd:
            run([py, "-m", "pip", "install", "-q", "pytest"], env=env)
        run([py, "-m", "pip", "install", "-q", "."], cwd=path, env=env)
        out = run(p.test_cmd.split(), cwd=path, env=env).stdout
        tests = count(r"(\d+) passed", out) if "pytest" in p.test_cmd else count(r"^Ran (\d+) tests?", out)
        if "unittest" in p.test_cmd and not re.search(r"^OK\b", out, re.M):
            raise GateFail("unittest did not report OK")
        binary = venv / "bin" / p.bin
    elif p.kind == "node":
        run(["npm", "ci" if p.extra["lock"] else "install", "--no-audit", "--no-fund"], cwd=path)
        out = run(["npm", "test"], cwd=path).stdout
        tests = max(count(r"^# pass (\d+)", out), count(r"^\S* ?pass (\d+)$", out),
                    count(r"Tests:.*?(\d+) passed", out),
                    count(r"(\d+) passing", out))
        binary = None
    else:
        raise GateFail(f"unknown kind {p.kind}")

    if tests < MIN_TESTS:
        raise GateFail(f"only {tests} passing tests (need {MIN_TESTS})")

    smoke = ["node", path / p.entry, "--help"] if binary is None else [binary, "--help"]
    out = run(smoke, cwd=scratch, env=env, timeout=60).stdout
    if not out.strip():
        raise GateFail("`--help` printed nothing")
    return tests


# ---------------------------------------------------------------- shaping the new repo

def git(args, cwd, **kw):
    return run(["git", *args], cwd=cwd, **kw)


def commit(repo: Path, message: str, paths: list[str]) -> None:
    git(["add", "--", *paths], cwd=repo)
    if git(["diff", "--cached", "--quiet"], cwd=repo, check=False, quiet=True).returncode != 0:
        git(["commit", "-q", "-m", message], cwd=repo)


def extract(root: Path, tool: str, owner: str, dest: Path) -> None:
    """Copy the folder's history into a fresh repo at dest, with #N refs pointing back at revived."""
    branch = f"spinout-split-{tool}"
    git(["branch", "-D", branch], cwd=root, check=False, quiet=True)
    git(["subtree", "split", "-q", f"--prefix={tool}", "-b", branch], cwd=root)
    git(["clone", "-q", "--no-local", "--branch", branch, str(root), str(dest)], cwd=root)
    git(["branch", "-D", branch], cwd=root, check=False, quiet=True)
    git(["checkout", "-q", "-B", "main"], cwd=dest)
    git(["remote", "remove", "origin"], cwd=dest)
    env = dict(os.environ, FILTER_BRANCH_SQUELCH_WARNING="1")
    msg_filter = f"sed -E 's@(^|[^A-Za-z0-9_/.-])#([0-9]+)@\\1{owner}/revived#\\2@g'"
    git(["filter-branch", "-f", "--msg-filter", msg_filter, "--", "main"], cwd=dest, env=env)
    for ref in run(["git", "for-each-ref", "--format=%(refname)", "refs/original"], cwd=dest, quiet=True).stdout.split():
        git(["update-ref", "-d", ref], cwd=dest, quiet=True)
    git(["config", "user.name", os.environ.get("GIT_AUTHOR_NAME", "Jaymz")], cwd=dest)
    git(["config", "user.email", os.environ.get("GIT_AUTHOR_EMAIL", "253701092+ag0nyy-ops@users.noreply.github.com")], cwd=dest)


def relicense(p: Project, repo: Path) -> None:
    shutil.copyfile(LICENSE_FILE, repo / "LICENSE")
    notice = NOTICE_FILE.read_text(encoding="utf-8").strip()
    readme_path = repo / "README.md"
    readme = readme_path.read_text(encoding="utf-8")
    if "view and learn only" not in readme:
        lines = readme.splitlines()
        at = next((i for i, l in enumerate(lines) if l.startswith("# ")), -1) + 1
        lines[at:at] = ["", notice]
        readme = "\n".join(lines) + "\n"
    # a "## License" section that says MIT would contradict the LICENSE file
    readme = re.sub(
        r"(^## License[^\n]*\n)(.*?)(?=^## |\Z)",
        lambda m: m.group(1) + "\nSource-available, view and learn only. See [LICENSE](LICENSE). "
        f"The original, [{p.original}](https://github.com/{p.original}), keeps its own license and credits.\n\n",
        readme, flags=re.S | re.M,
    )
    readme_path.write_text(readme, encoding="utf-8")

    touched = ["LICENSE", "README.md"]
    if p.kind == "rust":
        f = repo / "Cargo.toml"
        f.write_text(re.sub(r'^license\s*=.*$', 'license-file = "LICENSE"', f.read_text(encoding="utf-8"), flags=re.M), encoding="utf-8")
        touched.append("Cargo.toml")
    elif p.kind == "python":
        f = repo / "pyproject.toml"
        f.write_text(re.sub(r'^license\s*=.*$', 'license = { file = "LICENSE" }', f.read_text(encoding="utf-8"), flags=re.M), encoding="utf-8")
        touched.append("pyproject.toml")
    elif p.kind == "node":
        f = repo / "package.json"
        pkg = json.loads(f.read_text(encoding="utf-8"))
        pkg["license"] = "SEE LICENSE IN LICENSE"
        f.write_text(json.dumps(pkg, indent=2) + "\n", encoding="utf-8")
        touched.append("package.json")
    commit(repo, "license: source-available, view and learn only", touched)


def fix_go_module(p: Project, repo: Path, owner: str) -> None:
    old = p.extra.get("module", "")
    new = f"github.com/{owner}/{p.tool}"
    if not old or old == new:
        return
    changed = []
    for f in [repo / "go.mod", repo / "README.md", *repo.rglob("*.go")]:
        text = f.read_text(encoding="utf-8")
        updated = re.sub(rf'(?<![\w/.-]){re.escape(old)}(?=["/\s@`]|$)', new, text, flags=re.M)
        if updated != text:
            f.write_text(updated, encoding="utf-8")
            changed.append(str(f.relative_to(repo)))
    commit(repo, "go: module path for the standalone repo", changed)


def add_release_workflow(p: Project, repo: Path) -> None:
    text = (TEMPLATES / f"release-{p.kind}.yml").read_text(encoding="utf-8")
    text = (text.replace("__BIN__", p.bin)
                .replace("__ENTRY__", p.entry)
                .replace("__MAIN__", p.extra.get("main", "."))
                .replace("__LOCKED__", "--locked" if p.extra.get("locked") else "")
                .replace("__TESTCMD__", p.test_cmd)
                .replace("__INSTALL__", "npm ci" if p.extra.get("lock") else "npm install"))
    out = repo / ".github" / "workflows" / "release.yml"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(text, encoding="utf-8")
    commit(repo, "ci: build release binaries when a release is published", [".github/workflows/release.yml"])


def describe(p: Project, repo: Path) -> str:
    if p.description:
        d = p.description.strip().rstrip(".")
    else:
        paras = [b.strip() for b in (repo / "README.md").read_text(encoding="utf-8").split("\n\n")]
        d = next((b for b in paras if b and not b.startswith(("#", ">", "|", "-", "`", "["))), p.tool)
        d = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", d).replace("`", "").splitlines()[0].rstrip(".")
    if p.original and p.original.lower() not in d.lower():
        d += f", rebuilt from {p.original}"
    return d[:300]


def topics(p: Project) -> list[str]:
    lang = {"rust": "rust", "go": "golang", "python": "python", "node": "nodejs"}[p.kind]
    return [lang, "cli", "command-line-tool", "revived"]


def release_notes(p: Project, owner: str, tests: int) -> str:
    install = {
        "rust": "Download the archive for your platform below, unpack it and put the binary on your PATH.",
        "go": "Download the archive for your platform below, unpack it and put the binary on your PATH.",
        "python": f"Download `{p.bin}.pyz` below and run it with `python3 {p.bin}.pyz`, or install the wheel with `pip install`.",
        "node": "Download the `.tgz` below and install it with `npm install -g <file>`.",
    }[p.kind]
    return (
        f"First standalone release of {p.tool}, a from-scratch rebuild of "
        f"[{p.original}](https://github.com/{p.original}).\n\n"
        f"{install}\n\n"
        f"{tests} tests pass on this release. It started life in "
        f"[{owner}/revived](https://github.com/{owner}/revived/tree/main/{p.tool}), where its history and issues live.\n\n"
        f"Pre-release: expect rough edges and say so in an issue."
    )


def publish(p: Project, repo: Path, owner: str, tests: int) -> str:
    name = p.tool
    _, created = gh_api("POST", "/user/repos", {
        "name": name, "description": describe(p, repo), "private": False,
        "has_wiki": False, "has_projects": False, "auto_init": False,
    })
    url = created["html_url"]
    log(f"created {url}")
    gh_api("PUT", f"/repos/{owner}/{name}/topics", {"names": topics(p)}, check=False)
    gh_api("PUT", f"/repos/{owner}/{name}/actions/permissions/workflow",
           {"default_workflow_permissions": "write", "can_approve_pull_request_reviews": False}, check=False)

    token = os.environ["SPINOUT_TOKEN"]
    header = "AUTHORIZATION: basic " + base64.b64encode(f"x-access-token:{token}".encode()).decode()
    proc = subprocess.run(["git", "-c", f"http.https://github.com/.extraheader={header}",
                           "push", "-q", f"https://github.com/{owner}/{name}.git", "main:main"],
                          cwd=repo, text=True, capture_output=True)
    log("$ git push https://github.com/%s/%s.git main:main" % (owner, name))
    if proc.returncode != 0:
        raise RuntimeError(f"push failed: {proc.stderr.replace(token, '***')}")

    gh_api("POST", f"/repos/{owner}/{name}/releases", {
        "tag_name": VERSION, "target_commitish": "main", "name": f"{name} {VERSION}",
        "body": release_notes(p, owner, tests), "prerelease": True,
    })
    return url


# ---------------------------------------------------------------- main

def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--owner", default=os.environ.get("GITHUB_REPOSITORY_OWNER", "ag0nyy-ops"))
    ap.add_argument("--root", default=".", help="revived checkout")
    ap.add_argument("--tool", default="", help="only consider this folder")
    ap.add_argument("--dry-run", action="store_true", help="run the gate on every candidate, create nothing")
    ap.add_argument("--keep", action="store_true", help="keep the prepared repos (with --dry-run)")
    args = ap.parse_args()

    root = Path(args.root).resolve()
    owner = args.owner
    folders = tool_folders(root)
    if args.tool:
        if args.tool not in folders:
            log(f"no tool folder called {args.tool}")
            return 1
        folders = [args.tool]

    try:
        repos = existing_repos(owner)
    except RuntimeError as e:
        if not args.dry_run:
            raise
        log(f"warning: {e}; skipping the existing-repo check")
        repos = []
    names = {norm(r["name"]) for r in repos}
    blurbs = [(r["name"], (r.get("description") or "").lower()) for r in repos]

    summary(f"## spin-out {'dry run' if args.dry_run else 'run'}\n\n| tool | result |\n|---|---|")
    workdir = Path(tempfile.mkdtemp(prefix="spinout-"))
    log(f"candidates (oldest first): {', '.join(folders) or 'none'}")
    for tool in folders:
        log(f"\n=== {tool} ===")
        try:
            if norm(tool) in names:
                raise GateFail("already has its own repo")
            p = detect(tool, root / tool, owner)
            if p.kind == "python" and norm(p.extra.get("pkg_name", "")) in names:
                raise GateFail(f"a repo named like its package ({p.extra['pkg_name']}) exists")
            if p.original:
                dup = next((n for n, d in blurbs if p.original.lower() in d), None)
                if dup:
                    raise GateFail(f"{dup} already rebuilds {p.original}")
            check_readme(p, root / tool)

            repo = workdir / tool / "repo"
            extract(root, tool, owner, repo)
            if p.kind == "go":
                fix_go_module(p, repo, owner)
            relicense(p, repo)
            add_release_workflow(p, repo)

            build = workdir / tool / "build"
            shutil.copytree(repo, build, ignore=shutil.ignore_patterns(".git"))
            tests = gate(p, build, workdir / tool)
        except GateFail as e:
            first = str(e).splitlines()[0]
            log(f"skip {tool}: {e}")
            summary(f"| {tool} | skipped: {first.replace('|', '/')} |")
            continue

        if args.dry_run:
            log(f"PASS {tool}: {p.kind}, `{p.bin}`, {tests} tests, original {p.original}")
            log(f"  prepared repo: {repo}")
            summary(f"| {tool} | passes the gate ({tests} tests), would become {owner}/{tool} |")
            continue

        url = publish(p, repo, owner, tests)
        summary(f"| {tool} | spun out to {url} with a {VERSION} pre-release |")
        log(f"done: {url}")
        return 0

    if not args.keep:
        shutil.rmtree(workdir, ignore_errors=True)
    log("\nnothing spun out" if not args.dry_run else "\ndry run finished")
    return 0


if __name__ == "__main__":
    sys.exit(main())
