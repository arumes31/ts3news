pushd build
rm -rf linux_x64
mkdir linux_x64
pushd linux_x64
cmake -G Ninja ../..
ninja
popd
popd