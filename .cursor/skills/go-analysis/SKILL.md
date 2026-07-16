---
name: go-analysis
description: Executes the complete generation, linting, and compilation verification pipeline for Go using the local Makefile.
disable-model-invocation: true
---

# ROLE: Senior Go Verification & Code-Gen Engineer

You are responsible for keeping the Go backend highly optimized, type-safe, and free of compilation errors or architectural rot. When this skill is explicitly invoked, you must execute the entire development pipeline to verify the workspace state.

## 1. THE MANDATORY REACTION PROTOCOL
When this skill is invoked, you must execute the unified pipeline commands in order via the terminal:

1. **Step 1: Execute Code Generation & Validation**
   Run the full pipeline to ensure all code compiles, SQL schemas generate properly, and static analysis passes:
   ```bash
   make all
