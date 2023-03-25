rem install NuGet packages
nuget install installer\windows\Storx\packages.config -o installer\windows\packages
nuget install installer\windows\StorxTests\packages.config -o installer\windows\packages

rem build the test project
msbuild installer\windows\StorxTests

rem run the unit tests
vstest.console installer\windows\StorxTests\bin\Debug\StorxTests.dll
