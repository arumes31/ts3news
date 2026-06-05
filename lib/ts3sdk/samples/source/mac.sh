pushd build
rm -rf mac
mkdir mac
pushd mac
cmake -G Ninja ../..
ninja
popd
popd