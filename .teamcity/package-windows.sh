VERSION=$(git describe --tags --always --match "[0-9][0-9][0-9][0-9].*.*")
echo $VERSION

export TARGET_OS=windows
# This controls the directory the built artifacts go into
export BUILT_ARTIFACT_DIR=built_artifacts/
export FINAL_ARTIFACT_DIR=artifacts/
mkdir -p $BUILT_ARTIFACT_DIR
mkdir -p $FINAL_ARTIFACT_DIR
windowsArchs=("amd64" "386")
for arch in ${windowsArchs[@]}; do
    export TARGET_ARCH=$arch
    # Copy exe into final directory
    cp $BUILT_ARTIFACT_DIR/tunnellink-windows-$arch.exe ./tunnellink.exe
    make tunnellink-msi
    # Copy msi into final directory
    mv tunnellink-$VERSION-$arch.msi $FINAL_ARTIFACT_DIR/tunnellink-windows-$arch.msi
    cp $BUILT_ARTIFACT_DIR/tunnellink-windows-$arch.exe $FINAL_ARTIFACT_DIR/tunnellink-windows-$arch.exe 
done
