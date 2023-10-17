
# Contributing to Houston

Pull requests are always welcome. Note that a code of conduct applies to all spaces managed in the Houston project, 
including issues and pull requests: [Code of Conduct](./code_of_conduct.md)

Please see the [Developer Guide](./developer_guide.md) for tips and how-tos. 

When submitting a pull request, we ask you to check the following:

1. Documentation
   - Make sure all functionality is documented in a markdown file in the [docs/](./) directory.

2. Unit tests
   - Write a unit test in the relevant module to fully test the new functionality.

3. Code style & formatting
   - Run `go fmt` and commit before pushing.

4. Your code will be licensed under Houston's license, https://github.com/datasparq-ai/houston/blob/main/LICENSE.
   Make sure, if you have used existing code, that the license is compatible and include the license information in the contributed files, 
   or obtain permission from the original author to relicense the contributed code.

## Get Started

To clone the repo to your go src folder, run: 

```bash
go install github.com/datasparq-ai/houston
```

Then change directory to the Houston project:

```bash
cd $GOPATH/src/datasparq-ai/houston
```

Create a new branch for development:

```bash
git checkout -b feature/my-feature-name 
```

When you've finished work on your branch, please make a pull request directly to `main` branch.
