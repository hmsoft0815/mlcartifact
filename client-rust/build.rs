fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../proto/artifact.proto");
    tonic_prost_build::compile_protos("../proto/artifact.proto")?;
    Ok(())
}
