// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package executor_test

import (
	"strings"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb"
	"github.com/pingcap/tidb/plan"
	"github.com/pingcap/tidb/privilege/privileges"
	"github.com/pingcap/tidb/sessionctx/variable"
	"github.com/pingcap/tidb/util/auth"
	"github.com/pingcap/tidb/util/testkit"
	"github.com/pingcap/tidb/util/testutil"
)

func (s *testSuite) TestShow(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("use test")

	testSQL := `drop table if exists show_test`
	tk.MustExec(testSQL)
	testSQL = `create table SHOW_test (id int PRIMARY KEY AUTO_INCREMENT, c1 int comment "c1_comment", c2 int, c3 int default 1) ENGINE=InnoDB AUTO_INCREMENT=28934 DEFAULT CHARSET=utf8 COMMENT "table_comment";`
	tk.MustExec(testSQL)

	testSQL = "show columns from show_test;"
	result := tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 4)

	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row := result.Rows()[0]
	// For issue https://github.com/pingcap/tidb/issues/1061
	expectedRow := []interface{}{
		"SHOW_test", "CREATE TABLE `SHOW_test` (\n  `id` int(11) NOT NULL AUTO_INCREMENT,\n  `c1` int(11) DEFAULT NULL COMMENT 'c1_comment',\n  `c2` int(11) DEFAULT NULL,\n  `c3` int(11) DEFAULT '1',\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=28934 COMMENT='table_comment'"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	// For issue https://github.com/pingcap/tidb/issues/1918
	testSQL = `create table ptest(
		a int primary key,
		b double NOT NULL DEFAULT 2.0,
		c varchar(10) NOT NULL,
		d time unique,
		e timestamp NULL
	);`
	tk.MustExec(testSQL)
	testSQL = "show create table ptest;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"ptest", "CREATE TABLE `ptest` (\n  `a` int(11) NOT NULL,\n  `b` double NOT NULL DEFAULT '2.0',\n  `c` varchar(10) NOT NULL,\n  `d` time DEFAULT NULL,\n  `e` timestamp NULL DEFAULT NULL,\n  PRIMARY KEY (`a`),\n  UNIQUE KEY `d` (`d`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	// Issue #4684.
	tk.MustExec("drop table if exists `t1`")
	testSQL = "create table `t1` (" +
		"`c1` tinyint unsigned default null," +
		"`c2` smallint unsigned default null," +
		"`c3` mediumint unsigned default null," +
		"`c4` int unsigned default null," +
		"`c5` bigint unsigned default null);`"

	tk.MustExec(testSQL)
	testSQL = "show create table t1"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"t1", "CREATE TABLE `t1` (\n" +
			"  `c1` tinyint(3) UNSIGNED DEFAULT NULL,\n" +
			"  `c2` smallint(5) UNSIGNED DEFAULT NULL,\n" +
			"  `c3` mediumint(8) UNSIGNED DEFAULT NULL,\n" +
			"  `c4` int(10) UNSIGNED DEFAULT NULL,\n" +
			"  `c5` bigint(20) UNSIGNED DEFAULT NULL\n" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	testSQL = "SHOW VARIABLES LIKE 'character_set_results';"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)

	// Test case for index type and comment
	tk.MustExec(`create table show_index (id int, c int, primary key (id), index cIdx using hash (c) comment "index_comment_for_cIdx");`)
	tk.MustExec(`create index idx1 on show_index (id) using hash;`)
	tk.MustExec(`create index idx2 on show_index (id) comment 'idx';`)
	tk.MustExec(`create index idx3 on show_index (id) using hash comment 'idx';`)
	tk.MustExec(`alter table show_index add index idx4 (id) using btree comment 'idx';`)
	tk.MustExec(`create index idx5 using hash on show_index (id) using btree comment 'idx';`)
	tk.MustExec(`create index idx6 using hash on show_index (id);`)
	tk.MustExec(`create index idx7 on show_index (id);`)
	testSQL = "SHOW index from show_index;"
	tk.MustQuery(testSQL).Check(testutil.RowsWithSep("|",
		"show_index|0|PRIMARY|1|id|A|0|<nil>|<nil>||BTREE||",
		"show_index|1|cIdx|1|c|A|0|<nil>|<nil>|YES|HASH||index_comment_for_cIdx",
		"show_index|1|idx1|1|id|A|0|<nil>|<nil>|YES|HASH||",
		"show_index|1|idx2|1|id|A|0|<nil>|<nil>|YES|BTREE||idx",
		"show_index|1|idx3|1|id|A|0|<nil>|<nil>|YES|HASH||idx",
		"show_index|1|idx4|1|id|A|0|<nil>|<nil>|YES|BTREE||idx",
		"show_index|1|idx5|1|id|A|0|<nil>|<nil>|YES|BTREE||idx",
		"show_index|1|idx6|1|id|A|0|<nil>|<nil>|YES|HASH||",
		"show_index|1|idx7|1|id|A|0|<nil>|<nil>|YES|BTREE||",
	))

	// For show like with escape
	testSQL = `show tables like 'SHOW\_test'`
	result = tk.MustQuery(testSQL)
	rows := result.Rows()
	c.Check(rows, HasLen, 1)
	c.Check(rows[0], DeepEquals, []interface{}{"SHOW_test"})

	var ss stats
	variable.RegisterStatistics(ss)
	testSQL = "show status like 'character_set_results';"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), NotNil)

	tk.MustQuery("SHOW PROCEDURE STATUS WHERE Db='test'").Check(testkit.Rows())
	tk.MustQuery("SHOW TRIGGERS WHERE `Trigger` ='test'").Check(testkit.Rows())
	tk.MustQuery("SHOW PROCESSLIST;").Check(testkit.Rows())
	tk.MustQuery("SHOW EVENTS WHERE Db = 'test'").Check(testkit.Rows())
	tk.MustQuery("SHOW PLUGINS").Check(testkit.Rows())

	// Test show create database
	testSQL = `create database show_test_DB`
	tk.MustExec(testSQL)
	testSQL = "show create database show_test_DB;"
	tk.MustQuery(testSQL).Check(testutil.RowsWithSep("|",
		"show_test_DB|CREATE DATABASE `show_test_DB` /* !40100 DEFAULT CHARACTER SET utf8 */",
	))

	tk.MustExec("use show_test_DB")
	result = tk.MustQuery("SHOW index from show_index from test where Column_name = 'c'")
	c.Check(result.Rows(), HasLen, 1)

	// Test show full columns
	// for issue https://github.com/pingcap/tidb/issues/4224
	tk.MustExec(`drop table if exists show_test_comment`)
	tk.MustExec(`create table show_test_comment (id int not null default 0 comment "show_test_comment_id")`)
	tk.MustQuery(`show full columns from show_test_comment`).Check(testutil.RowsWithSep("|",
		"id|int(11)|binary|NO||0||select,insert,update,references|show_test_comment_id",
	))

	// Test show create table with AUTO_INCREMENT option
	// for issue https://github.com/pingcap/tidb/issues/3747
	tk.MustExec(`drop table if exists show_auto_increment`)
	tk.MustExec(`create table show_auto_increment (id int) auto_increment=4`)
	tk.MustQuery(`show create table show_auto_increment`).Check(testutil.RowsWithSep("|",
		""+
			"show_auto_increment CREATE TABLE `show_auto_increment` (\n"+
			"  `id` int(11) DEFAULT NULL\n"+
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=4",
	))

	// Test show table with column's comment contain escape character
	// for issue https://github.com/pingcap/tidb/issues/4411
	tk.MustExec(`drop table if exists show_escape_character`)
	tk.MustExec(`create table show_escape_character(id int comment 'a\rb\nc\td\0ef')`)
	tk.MustQuery(`show create table show_escape_character`).Check(testutil.RowsWithSep("|",
		""+
			"show_escape_character CREATE TABLE `show_escape_character` (\n"+
			"  `id` int(11) DEFAULT NULL COMMENT 'a\\rb\\nc	d\\0ef'\n"+
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin",
	))

	// for issue https://github.com/pingcap/tidb/issues/4424
	tk.MustExec("drop table if exists show_test")
	testSQL = `create table show_test(
		a varchar(10) COMMENT 'a\nb\rc\td\0e'
	) COMMENT='a\nb\rc\td\0e';`
	tk.MustExec(testSQL)
	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"show_test", "CREATE TABLE `show_test` (\n  `a` varchar(10) DEFAULT NULL COMMENT 'a\\nb\\rc	d\\0e'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin COMMENT='a\\nb\\rc	d\\0e'"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	// for issue https://github.com/pingcap/tidb/issues/4425
	tk.MustExec("drop table if exists show_test")
	testSQL = `create table show_test(
		a varchar(10) DEFAULT 'a\nb\rc\td\0e'
	);`
	tk.MustExec(testSQL)
	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"show_test", "CREATE TABLE `show_test` (\n  `a` varchar(10) DEFAULT 'a\\nb\\rc	d\\0e'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	// for issue https://github.com/pingcap/tidb/issues/4426
	tk.MustExec("drop table if exists show_test")
	testSQL = `create table show_test(
		a bit(1),
		b bit(32) DEFAULT 0b0,
		c bit(1) DEFAULT 0b1,
		d bit(10) DEFAULT 0b1010
	);`
	tk.MustExec(testSQL)
	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"show_test", "CREATE TABLE `show_test` (\n  `a` bit(1) DEFAULT NULL,\n  `b` bit(32) DEFAULT b'0',\n  `c` bit(1) DEFAULT b'1',\n  `d` bit(10) DEFAULT b'1010'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}
}

func (s *testSuite) TestShowVisibility(c *C) {
	save := privileges.Enable
	privileges.Enable = true
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("create database showdatabase")
	tk.MustExec("use showdatabase")
	tk.MustExec("create table t1 (id int)")
	tk.MustExec("create table t2 (id int)")
	tk.MustExec(`create user 'show'@'%'`)
	tk.MustExec(`flush privileges`)

	tk1 := testkit.NewTestKit(c, s.store)
	se, err := tidb.CreateSession(s.store)
	c.Assert(err, IsNil)
	c.Assert(se.Auth(&auth.UserIdentity{Username: "show", Hostname: "%"}, nil, nil), IsTrue)
	tk1.Se = se

	// No ShowDatabases privilege, this user would see nothing except INFORMATION_SCHEMA.
	tk.MustQuery("show databases").Check(testkit.Rows("INFORMATION_SCHEMA"))

	// After grant, the user can see the database.
	tk.MustExec(`grant select on showdatabase.t1 to 'show'@'%'`)
	tk.MustExec(`flush privileges`)
	tk1.MustQuery("show databases").Check(testkit.Rows("INFORMATION_SCHEMA", "showdatabase"))

	// The user can see t1 but not t2.
	tk1.MustExec("use showdatabase")
	tk1.MustQuery("show tables").Check(testkit.Rows("t1"))

	// After revoke, show database result should be just except INFORMATION_SCHEMA.
	tk.MustExec(`revoke select on showdatabase.t1 from 'show'@'%'`)
	tk.MustExec(`flush privileges`)
	tk1.MustQuery("show databases").Check(testkit.Rows("INFORMATION_SCHEMA"))

	// Grant any global privilege would make show databases available.
	tk.MustExec(`grant CREATE on *.* to 'show'@'%'`)
	tk.MustExec(`flush privileges`)
	rows := tk1.MustQuery("show databases").Rows()
	c.Assert(len(rows), GreaterEqual, 2) // At least INFORMATION_SCHEMA and showdatabase

	privileges.Enable = save
	tk.MustExec(`drop user 'show'@'%'`)
	tk.MustExec("drop database showdatabase")
}

type stats struct {
}

func (s stats) GetScope(status string) variable.ScopeFlag { return variable.DefaultStatusVarScopeFlag }

func (s stats) Stats(vars *variable.SessionVars) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	var a, b interface{}
	b = "123"
	m["test_interface_nil"] = a
	m["test_interface"] = b
	m["test_interface_slice"] = []interface{}{"a", "b", "c"}
	return m, nil
}

func (s *testSuite) TestForeignKeyInShowCreateTable(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("use test")
	testSQL := `drop table if exists show_test`
	tk.MustExec(testSQL)
	testSQL = `drop table if exists t1`
	tk.MustExec(testSQL)
	testSQL = `CREATE TABLE t1 (pk int PRIMARY KEY AUTO_INCREMENT)`
	tk.MustExec(testSQL)

	// For table with single fk.
	sqlLines := []string{
		"CREATE TABLE `show_test` (",
		"  `id` int(11) NOT NULL AUTO_INCREMENT,",
		"  PRIMARY KEY (`id`),",
		"  CONSTRAINT `Fk` FOREIGN KEY (`id`) REFERENCES `t1` (`pk`) ON DELETE CASCADE ON UPDATE CASCADE",
		") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin",
	}
	testSQL = strings.Join(sqlLines, "\n")
	tk.MustExec(testSQL)
	result := tk.MustQuery("show create table show_test;")
	c.Check(result.Rows(), HasLen, 1)
	row := result.Rows()[0]
	expectedRow := []interface{}{"show_test", testSQL}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	// For table with multiple fk.
	sqlLines = []string{
		"CREATE TABLE `pilot_languages` (",
		"  `pilot_id` int(11) NOT NULL,",
		"  `language_id` int(11) NOT NULL,",
		"  CONSTRAINT `pilot_language_fkey` FOREIGN KEY (`pilot_id`) REFERENCES `pilots` (`pilot_id`),",
		"  CONSTRAINT `languages_fkey` FOREIGN KEY (`language_id`) REFERENCES `languages` (`language_id`)",
		") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin",
	}
	testSQL = strings.Join(sqlLines, "\n")
	tk.MustExec(testSQL)
	result = tk.MustQuery("show create table pilot_languages;")
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{"pilot_languages", testSQL}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	testSQL = "alter table show_test drop foreign key `fk`"
	tk.MustExec(testSQL)
	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"show_test", "CREATE TABLE `show_test` (\n  `id` int(11) NOT NULL AUTO_INCREMENT,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}

	testSQL = "ALTER TABLE SHOW_TEST ADD CONSTRAINT `Fk` FOREIGN KEY (`id`) REFERENCES `t1` (`pk`) ON DELETE CASCADE ON UPDATE CASCADE\n "
	tk.MustExec(testSQL)
	testSQL = "show create table show_test;"
	result = tk.MustQuery(testSQL)
	c.Check(result.Rows(), HasLen, 1)
	row = result.Rows()[0]
	expectedRow = []interface{}{
		"show_test", "CREATE TABLE `show_test` (\n  `id` int(11) NOT NULL AUTO_INCREMENT,\n  PRIMARY KEY (`id`),\n  CONSTRAINT `Fk` FOREIGN KEY (`id`) REFERENCES `t1` (`pk`) ON DELETE CASCADE ON UPDATE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"}
	for i, r := range row {
		c.Check(r, Equals, expectedRow[i])
	}
}

func (s *testSuite) TestShowWarnings(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("use test")
	testSQL := `create table if not exists show_warnings (a int)`
	tk.MustExec(testSQL)
	tk.MustExec("set @@sql_mode=''")
	tk.MustExec("insert show_warnings values ('a')")
	c.Assert(tk.Se.GetSessionVars().StmtCtx.WarningCount(), Equals, uint16(1))
	tk.MustQuery("show warnings").Check(testutil.RowsWithSep("|", "Warning|1265|Data Truncated"))
	c.Assert(tk.Se.GetSessionVars().StmtCtx.WarningCount(), Equals, uint16(0))
	tk.MustQuery("show warnings").Check(testutil.RowsWithSep("|", "Warning|1265|Data Truncated"))
	c.Assert(tk.Se.GetSessionVars().StmtCtx.WarningCount(), Equals, uint16(0))
}

func (s *testSuite) TestIssue3641(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	_, err := tk.Exec("show tables;")
	c.Assert(err.Error(), Equals, plan.ErrNoDB.Error())
	_, err = tk.Exec("show table status;")
	c.Assert(err.Error(), Equals, plan.ErrNoDB.Error())
}

// TestShow2 is moved from session_test
func (s *testSuite) TestShow2(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("use test")

	tk.MustExec("set global autocommit=0")
	tk1 := testkit.NewTestKit(c, s.store)
	tk1.MustQuery("show global variables where variable_name = 'autocommit'").Check(testkit.Rows("autocommit 0"))
	tk.MustExec("set global autocommit = 1")
	tk2 := testkit.NewTestKit(c, s.store)
	// TODO: In MySQL, the result is "autocommit ON".
	tk2.MustQuery("show global variables where variable_name = 'autocommit'").Check(testkit.Rows("autocommit 1"))

	tk.MustExec("drop table if exists t")
	tk.MustExec(`create table if not exists t (c int) comment '注释'`)
	tk.MustQuery(`show columns from t`).Check(testutil.RowsWithSep(",", "c,int(11),YES,,<nil>,"))

	tk.MustQuery("show collation where Charset = 'utf8' and Collation = 'utf8_bin'").Check(testutil.RowsWithSep(",", "utf8_bin,utf8,83,,Yes,1"))

	tk.MustQuery("show tables").Check(testkit.Rows("t"))
	tk.MustQuery("show full tables").Check(testkit.Rows("t BASE TABLE"))

	r, err := tk.Exec("show table status from test like 't'")
	c.Assert(err, IsNil)
	row, err := r.Next()
	c.Assert(err, IsNil)
	c.Assert(row.Data, HasLen, 18)
	c.Assert(row.Data[0].GetString(), Equals, "t")
	c.Assert(row.Data[17].GetString(), Equals, "注释")

	tk.Se.Auth(&auth.UserIdentity{Username: "root", Hostname: "%"}, nil, []byte("012345678901234567890"))

	tk.MustQuery("show databases like 'test'").Check(testkit.Rows("test"))

	tk.MustExec("grant all on *.* to 'root'@'%'")
	tk.MustQuery("show grants").Check(testkit.Rows("GRANT ALL PRIVILEGES ON *.* TO 'root'@'%'"))
}