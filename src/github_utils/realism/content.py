"""Content Realism Engine -- generates realistic commit messages and file changes.

Produces conventional-commit-style messages (e.g. "feat: add auth support")
from template banks of prefixes, components, and actions, yielding 200+
unique combinations. File changes span multiple extensions (.py, .js, .ts,
.md, .yaml, .json) with 1-4 files per commit, weighted toward fewer files.
"""

from __future__ import annotations

import random
from typing import Dict, List, Optional

# ---------------------------------------------------------------------------
# Commit message templates
# ---------------------------------------------------------------------------

# Conventional commit prefixes.
_PREFIXES: List[str] = [
    "feat",
    "fix",
    "refactor",
    "docs",
    "test",
    "chore",
    "style",
    "perf",
    "ci",
    "build",
]

# Software components that appear in commit messages.
_COMPONENTS: List[str] = [
    "auth",
    "api",
    "database",
    "cache",
    "logging",
    "config",
    "router",
    "middleware",
    "parser",
    "serializer",
    "validator",
    "scheduler",
    "worker",
    "pipeline",
    "dashboard",
    "notifications",
    "search",
    "analytics",
    "payments",
    "storage",
]

# Action phrases that follow the component reference.
_ACTIONS: List[str] = [
    "add {component} support",
    "update {component} logic",
    "fix {component} error handling",
    "refactor {component} module",
    "improve {component} performance",
    "remove deprecated {component} code",
    "add tests for {component}",
    "update {component} configuration",
    "simplify {component} interface",
    "handle edge case in {component}",
    "migrate {component} to async",
    "add retry logic to {component}",
]

# Total combinations: 10 prefixes * 20 components * 12 actions = 2400

# ---------------------------------------------------------------------------
# File change templates
# ---------------------------------------------------------------------------

# (directory, filename_stem_choices, extension)
_FILE_TEMPLATES: List[tuple] = [
    ("src", ["main", "app", "index", "server", "utils", "helpers", "config"], ".py"),
    ("src", ["handler", "service", "controller", "model", "schema"], ".py"),
    ("lib", ["client", "connection", "pool", "adapter", "wrapper"], ".py"),
    ("src", ["app", "index", "server", "utils", "helpers", "config"], ".js"),
    ("src/components", ["Button", "Header", "Footer", "Sidebar", "Modal"], ".tsx"),
    ("src", ["types", "interfaces", "constants", "utils", "helpers"], ".ts"),
    ("docs", ["README", "CHANGELOG", "CONTRIBUTING", "API", "SETUP"], ".md"),
    (".", ["config", "settings", "docker-compose", "ci"], ".yaml"),
    (".", ["package", "tsconfig", "schema", ".eslintrc"], ".json"),
    ("tests", ["test_main", "test_utils", "test_api", "test_models"], ".py"),
    ("scripts", ["deploy", "build", "migrate", "seed"], ".sh"),
    ("src/styles", ["main", "theme", "variables", "reset"], ".css"),
]

# Content snippets per extension, used to build realistic file bodies.
_CONTENT_SNIPPETS: Dict[str, List[str]] = {
    ".py": [
        'def {func}():\n    """Process input data."""\n    return None\n',
        "class {cls}:\n    def __init__(self):\n        self.data = {{}}\n",
        "import os\nimport sys\n\nVERSION = \"1.0.0\"\n",
        "from typing import List, Optional\n\ndef {func}(items: List[str]) -> Optional[str]:\n    return items[0] if items else None\n",
        "# Updated configuration\nDEBUG = False\nLOG_LEVEL = \"INFO\"\n",
    ],
    ".js": [
        "const {func} = () => {{\n  console.log('initialized');\n}};\n\nmodule.exports = {{ {func} }};\n",
        "import express from 'express';\n\nconst app = express();\napp.get('/', (req, res) => res.json({{ status: 'ok' }}));\n",
        "function {func}(data) {{\n  if (!data) throw new Error('missing data');\n  return data;\n}}\n",
    ],
    ".ts": [
        "export interface {cls} {{\n  id: string;\n  name: string;\n  createdAt: Date;\n}}\n",
        "export const {func} = (input: string): boolean => {{\n  return input.length > 0;\n}};\n",
        "type Config = {{\n  host: string;\n  port: number;\n  debug: boolean;\n}};\n\nexport const defaults: Config = {{\n  host: 'localhost',\n  port: 3000,\n  debug: false,\n}};\n",
    ],
    ".tsx": [
        "import React from 'react';\n\nexport const {cls}: React.FC = () => {{\n  return <div className=\"{func}\">Content</div>;\n}};\n",
        "import {{ useState }} from 'react';\n\nexport function {cls}() {{\n  const [open, setOpen] = useState(false);\n  return <button onClick={{() => setOpen(!open)}}>Toggle</button>;\n}}\n",
    ],
    ".md": [
        "# {cls}\n\nThis module handles {func} operations.\n\n## Usage\n\n```bash\nnpm install\nnpm start\n```\n",
        "## Changes\n\n- Updated {func} implementation\n- Fixed edge case in {cls}\n- Improved documentation\n",
        "# API Reference\n\n## `{func}()`\n\nReturns processed data.\n\n### Parameters\n\n| Name | Type | Description |\n|------|------|-------------|\n| data | object | Input data |\n",
    ],
    ".yaml": [
        "name: {func}\nversion: '1.0'\nservices:\n  app:\n    build: .\n    ports:\n      - '8080:8080'\n",
        "env: production\ndebug: false\nlog_level: info\ndatabase:\n  host: localhost\n  port: 5432\n",
    ],
    ".json": [
        '{{\n  "name": "{func}",\n  "version": "1.0.0",\n  "description": "A project module",\n  "main": "index.js"\n}}\n',
        '{{\n  "compilerOptions": {{\n    "target": "es2020",\n    "module": "commonjs",\n    "strict": true\n  }}\n}}\n',
    ],
    ".sh": [
        "#!/usr/bin/env bash\nset -euo pipefail\n\necho \"Running {func}...\"\n",
        "#!/bin/bash\n# {cls} script\ncd \"$(dirname \"$0\")\"\npython -m {func}\n",
    ],
    ".css": [
        ".{func} {{\n  display: flex;\n  align-items: center;\n  padding: 1rem;\n}}\n",
        ":root {{\n  --primary: #3b82f6;\n  --secondary: #10b981;\n  --font-size: 16px;\n}}\n",
    ],
}

# Identifier pools for template interpolation.
_FUNC_NAMES: List[str] = [
    "process", "handle", "validate", "transform", "parse",
    "serialize", "fetch", "compute", "render", "dispatch",
    "init", "cleanup", "migrate", "sync", "export",
]

_CLASS_NAMES: List[str] = [
    "Manager", "Service", "Handler", "Controller", "Factory",
    "Builder", "Adapter", "Provider", "Resolver", "Engine",
    "Gateway", "Repository", "Processor", "Validator", "Dispatcher",
]

# Weights for number of files per commit: heavily biased toward 1.
_FILE_COUNT_POPULATION = [1, 2, 3, 4]
_FILE_COUNT_WEIGHTS = [50, 30, 15, 5]


class ContentEngine:
    """Generate realistic commit messages and file changes.

    Parameters
    ----------
    seed:
        Optional integer seed for deterministic output via a private
        ``random.Random`` instance.
    """

    def __init__(self, seed: Optional[int] = None) -> None:
        self._rng = random.Random(seed)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def generate_message(self) -> str:
        """Return a conventional-commit-style commit message.

        Format: ``"<prefix>: <action with component>"``
        """
        prefix = self._rng.choice(_PREFIXES)
        component = self._rng.choice(_COMPONENTS)
        action_template = self._rng.choice(_ACTIONS)
        action = action_template.format(component=component)
        return f"{prefix}: {action}"

    def generate_file_changes(self) -> Dict[str, str]:
        """Return a dict mapping file paths to generated content.

        Produces 1-4 files per call (weighted toward 1). Each file has a
        realistic path and extension-appropriate content body.
        """
        count = self._rng.choices(
            population=_FILE_COUNT_POPULATION,
            weights=_FILE_COUNT_WEIGHTS,
            k=1,
        )[0]

        changes: Dict[str, str] = {}
        for _ in range(count):
            directory, stems, ext = self._rng.choice(_FILE_TEMPLATES)
            stem = self._rng.choice(stems)

            if directory == ".":
                path = f"{stem}{ext}"
            else:
                path = f"{directory}/{stem}{ext}"

            content = self._generate_content(ext)
            changes[path] = content

        return changes

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _generate_content(self, ext: str) -> str:
        """Build a content string appropriate for the given file extension."""
        snippets = _CONTENT_SNIPPETS.get(ext, _CONTENT_SNIPPETS[".py"])
        template = self._rng.choice(snippets)

        func = self._rng.choice(_FUNC_NAMES)
        cls = self._rng.choice(_CLASS_NAMES)

        return template.format(func=func, cls=cls)
