//! End-to-end test against a running artifact-server.
//! Skipped unless `MLCARTIFACT_E2E_ADDR` is set, e.g.
//! `MLCARTIFACT_E2E_ADDR=http://127.0.0.1:19593 cargo test`.

use mlcartifact::{ArtifactClient, ListOptions, WriteOptions};

#[tokio::test]
async fn vfs_roundtrip() {
    let Ok(addr) = std::env::var("MLCARTIFACT_E2E_ADDR") else {
        eprintln!("MLCARTIFACT_E2E_ADDR not set, skipping e2e test");
        return;
    };
    let c = ArtifactClient::connect(addr).await.expect("connect");
    let vpath = "/rust-e2e/docs/notes.txt".to_string();

    // write with virtual_path
    let w = c
        .write_file(
            "notes.txt",
            b"line0\nline1\nline2".to_vec(),
            WriteOptions {
                virtual_path: Some(vpath.clone()),
                source: Some("rust-e2e".into()),
                ..Default::default()
            },
        )
        .await
        .expect("write");
    assert!(!w.id.is_empty());
    assert_eq!(w.virtual_path, vpath);

    // read by id and by virtual path
    let r = c.read(w.id.clone(), None).await.expect("read by id");
    assert_eq!(r.content, b"line0\nline1\nline2");
    let r = c.read(vpath.clone(), None).await.expect("read by path");
    assert_eq!(r.virtual_path, vpath);

    // flat list (legacy signature)
    let l = c.list(None, None, None).await.expect("list");
    assert!(l.items.iter().any(|i| i.id == w.id));

    // list with dir_path
    let l = c
        .list_with(ListOptions {
            dir_path: Some("/rust-e2e".into()),
            ..Default::default()
        })
        .await
        .expect("list dir");
    assert!(
        l.items.iter().any(|i| i.is_directory && i.filename.contains("docs")),
        "expected docs dir in {:?}",
        l.items
    );
    let l = c
        .list_with(ListOptions {
            dir_path: Some("/rust-e2e/docs".into()),
            ..Default::default()
        })
        .await
        .expect("list subdir");
    assert!(l.items.iter().any(|i| i.id == w.id), "file missing in {:?}", l.items);

    // patch: replace line 1, then append
    let p = c
        .patch(vpath.clone(), b"LINE1".to_vec(), Some(1), Some(2), false, None)
        .await
        .expect("patch lines");
    assert!(p.success);
    let p = c
        .patch(vpath.clone(), b"\nline3".to_vec(), None, None, true, None)
        .await
        .expect("patch append");
    assert!(p.success);
    let r = c.read(w.id.clone(), None).await.expect("read after patch");
    assert_eq!(String::from_utf8_lossy(&r.content), "line0\nLINE1\nline2\nline3");
    assert_eq!(p.new_size as usize, r.content.len());

    // find
    let f = c.find("/rust-e2e/docs/*.txt", None).await.expect("find");
    assert!(f.items.iter().any(|i| i.id == w.id), "find missing: {:?}", f.items);

    // delete by path
    let d = c.delete(vpath.clone(), None).await.expect("delete");
    assert!(d.deleted);
    let err = c.read(w.id.clone(), None).await.expect_err("should be gone");
    assert_eq!(err.code(), tonic::Code::NotFound);
}
