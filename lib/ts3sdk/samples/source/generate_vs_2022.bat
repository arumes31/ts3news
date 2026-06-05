pushd build
pushd win_x64
cmake -G "Visual Studio 17 2022" -A x64 ../..
popd
popd
