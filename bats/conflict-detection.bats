#!/usr/bin/env bats

setup() {
    export PATH=$PATH:~/go/bin
    export NOMS_VERSION_NEXT=1
    cd $BATS_TMPDIR
    mkdir "dolt-repo-$$"
    cd "dolt-repo-$$"
    dolt init
}

teardown() {
    rm -rf "$BATS_TMPDIR/dolt-repo-$$"
}

@test "two branches modify different cell same row. merge. no conflict" {
    dolt table create -s=$BATS_TEST_DIRNAME/helper/1pk5col-ints.schema test
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "table created"
    dolt branch change-cell
    dolt table put-row test pk:0 c1:11 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "changed pk=0 c1 to 11"
    dolt checkout change-cell
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:11
    dolt add test
    dolt commit -m "changed pk=0 c5 to 11"
    dolt checkout master
    run dolt merge change-cell
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Updating" ]] || false
    [[ "$output" =~ "1 tables changed" ]] || false
    [[ "$output" =~ "1 rows modified" ]] || false
}

@test "two branches modify same cell. merge. conflict" {
    dolt table create -s=$BATS_TEST_DIRNAME/helper/1pk5col-ints.schema test
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "table created"
    dolt branch change-cell
    dolt table put-row test pk:0 c1:1 c2:1 c3:1 c4:1 c5:1
    dolt add test
    dolt commit -m "changed pk=0 all cells to 1"
    dolt checkout change-cell
    dolt table put-row test pk:0 c1:11 c2:11 c3:11 c4:11 c5:11
    dolt add test
    dolt commit -m "changed pk=0 all cells to 11"
    dolt checkout master
    run dolt merge change-cell
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CONFLICT" ]] || false
}

@test "two branches add a different row. merge. no conflict" {
    dolt table create -s=$BATS_TEST_DIRNAME/helper/1pk5col-ints.schema test
    dolt add test
    dolt commit -m "table created"
    dolt branch add-row
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "added pk=0 row"
    dolt checkout add-row
    dolt table put-row test pk:1 c1:1 c2:1 c3:1 c4:1 c5:1
    dolt add test
    dolt commit -m "added pk=1 row"
    dolt checkout master
    run dolt merge add-row
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Updating" ]] || false
    [[ "$output" =~ "1 tables changed" ]] || false
    [[ "$output" =~ "1 rows added" ]] || false
}

@test "two branches add same row. merge. no conflict" {
    dolt table create -s=$BATS_TEST_DIRNAME/helper/1pk5col-ints.schema test
    dolt add test
    dolt commit -m "table created"
    dolt branch add-row
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "added pk=0 row"
    dolt checkout add-row
    dolt table put-row test pk:0 c1:0 c2:0 c3:0 c4:0 c5:0
    dolt add test
    dolt commit -m "added pk=0 row"
    dolt checkout master
    run dolt merge add-row
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Updating" ]] || false
}

@test "two branches both create different tables. merge. no conflict" {
    dolt branch table1
    dolt branch table2
    dolt checkout table1
    dolt table create -s=$BATS_TEST_DIRNAME/helper/1pk5col-ints.schema table1
    dolt add table1
    dolt commit -m "first table"
    dolt checkout table2
    dolt table create -s=$BATS_TEST_DIRNAME/helper/2pk5col-ints.schema table2
    dolt add table2
    dolt commit -m "second table"
    dolt checkout master
    run dolt merge table1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Fast-forward" ]] || false
    run dolt merge table2
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Updating" ]] || false
}