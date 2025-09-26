# Publishing kport to code.nym.sh

The repository has been prepared and committed locally. To publish to https://code.nym.sh/kennethnym/kport.git, you'll need to push manually with authentication.

## Steps to publish:

1. **Ensure you have access to the repository**:
   ```bash
   # The remote is already configured
   git remote -v
   # Should show: origin  https://code.nym.sh/kennethnym/kport.git (fetch)
   #              origin  https://code.nym.sh/kennethnym/kport.git (push)
   ```

2. **Push to the repository**:
   ```bash
   git push -u origin main
   ```
   
   You'll be prompted for your username and password/token for code.nym.sh.

## Repository Contents:

- ✅ Complete kport application source code
- ✅ Comprehensive README.md with usage instructions
- ✅ Demo script (run_demo.sh)
- ✅ Go module configuration
- ✅ .gitignore file (properly excludes binaries)
- ✅ All commits properly attributed
- ✅ No binaries committed (build with `go build -o kport`)

## Current Status:

- Repository initialized: ✅
- Remote configured: ✅
- Files committed: ✅
- Binary properly ignored: ✅
- Screenshot added: ✅
- SSH command rewrite completed: ✅
- Ready to push: ✅

## Current Commits Ready to Push:

```
58c10d5 Add screenshot to README
9ec67e9 Replace Go SSH library with native ssh command
02322c4 Remove debug log file
bde1529 Fix port detection and manual forwarding issues
70307c7 Document quoted include support in README
a332459 Update README with new features
5ebe20a Improve manual port input UI styling
df3c9fe Add support for SSH config includes
6e1ee6d Add .gitignore file
e39a595 Initial commit: kport - SSH Port Forwarder TUI
```

## Push Issue:
Authentication required - push manually from local machine with credentials.

The application is fully functional and ready for distribution!