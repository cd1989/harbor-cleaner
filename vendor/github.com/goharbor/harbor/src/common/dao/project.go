// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils/log"

	"fmt"
	"time"
)

// AddProject adds a project to the database along with project roles information and access log records.
func AddProject(project models.Project) (int64, error) {
	o := GetOrmer()

	sql := "insert into project (owner_id, name, creation_time, update_time, deleted) values (?, ?, ?, ?, ?) RETURNING project_id"
	var projectID int64
	now := time.Now()

	err := o.Raw(sql, project.OwnerID, project.Name, now, now, project.Deleted).QueryRow(&projectID)
	if err != nil {
		return 0, err
	}

	pmID, err := addProjectMember(models.Member{
		ProjectID:  projectID,
		EntityID:   project.OwnerID,
		Role:       models.PROJECTADMIN,
		EntityType: common.UserMember,
	})
	if err != nil {
		return 0, err
	}
	if pmID == 0 {
		return projectID, err
	}
	return projectID, nil
}

func addProjectMember(member models.Member) (int, error) {

	log.Debugf("Adding project member %+v", member)

	o := GetOrmer()

	if member.EntityID <= 0 {
		return 0, fmt.Errorf("Invalid entity_id, member: %+v", member)
	}

	if member.ProjectID <= 0 {
		return 0, fmt.Errorf("Invalid project_id, member: %+v", member)
	}

	var pmID int
	sql := "insert into project_member (project_id, entity_id , role, entity_type) values (?, ?, ?, ?) RETURNING id"
	err := o.Raw(sql, member.ProjectID, member.EntityID, member.Role, member.EntityType).QueryRow(&pmID)
	if err != nil {
		return 0, err
	}
	return pmID, err
}

// GetProjectByID ...
func GetProjectByID(id int64) (*models.Project, error) {
	o := GetOrmer()

	sql := `select p.project_id, p.name, u.username as owner_name, p.owner_id, p.creation_time, p.update_time  
		from project p left join harbor_user u on p.owner_id = u.user_id where p.deleted = false and p.project_id = ?`
	queryParam := make([]interface{}, 1)
	queryParam = append(queryParam, id)

	p := []models.Project{}
	count, err := o.Raw(sql, queryParam).QueryRows(&p)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return &p[0], nil
}

// GetProjectByName ...
func GetProjectByName(name string) (*models.Project, error) {
	o := GetOrmer()
	var p []models.Project
	n, err := o.Raw(`select * from project where name = ? and deleted = false`, name).QueryRows(&p)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	return &p[0], nil
}

// ProjectExistsByName returns whether the project exists according to its name.
func ProjectExistsByName(name string) bool {
	o := GetOrmer()
	return o.QueryTable("project").Filter("name", name).Exist()
}

// GetTotalOfProjects returns the total count of projects
// according to the query conditions
func GetTotalOfProjects(query *models.ProjectQueryParam) (int64, error) {
	var pagination *models.Pagination
	if query != nil {
		pagination = query.Pagination
		query.Pagination = nil
	}
	sql, params := projectQueryConditions(query)
	if query != nil {
		query.Pagination = pagination
	}

	sql = `select count(*) ` + sql

	var total int64
	err := GetOrmer().Raw(sql, params).QueryRow(&total)
	return total, err
}

// GetProjects returns a project list according to the query conditions
func GetProjects(query *models.ProjectQueryParam) ([]*models.Project, error) {
	sqlStr, queryParam := projectQueryConditions(query)
	sqlStr = `select distinct p.project_id, p.name, p.owner_id, 
		p.creation_time, p.update_time ` + sqlStr + ` order by p.name`
	sqlStr, queryParam = CreatePagination(query, sqlStr, queryParam)

	log.Debugf("sql:=%+v, param= %+v", sqlStr, queryParam)
	var projects []*models.Project
	_, err := GetOrmer().Raw(sqlStr, queryParam).QueryRows(&projects)

	return projects, err

}

// GetGroupProjects - Get user's all projects, including user is the user member of this project
// and the user is in the group which is a group member of this project.
func GetGroupProjects(groupDNCondition string, query *models.ProjectQueryParam) ([]*models.Project, error) {
	sql, params := projectQueryConditions(query)
	sql = `select distinct p.project_id, p.name, p.owner_id, 
				p.creation_time, p.update_time ` + sql
	if len(groupDNCondition) > 0 {
		sql = fmt.Sprintf(
			`%s union select distinct p.project_id, p.name, p.owner_id, p.creation_time, p.update_time  
		     from project p 
		     left join project_member pm on p.project_id = pm.project_id
		     left join user_group ug on ug.id = pm.entity_id and pm.entity_type = 'g' and ug.group_type = 1
			 where ug.ldap_group_dn in ( %s ) order by name`,
			sql, groupDNCondition)
	}
	sqlStr, queryParams := CreatePagination(query, sql, params)
	log.Debugf("query sql:%v", sql)
	var projects []*models.Project
	_, err := GetOrmer().Raw(sqlStr, queryParams).QueryRows(&projects)
	return projects, err
}

// GetTotalGroupProjects - Get the total count of projects, including  user is the member of this project and the
// user is in the group, which is the group member of this project.
func GetTotalGroupProjects(groupDNCondition string, query *models.ProjectQueryParam) (int, error) {
	var sql string
	sqlCondition, params := projectQueryConditions(query)
	if len(groupDNCondition) == 0 {
		sql = `select count(1) ` + sqlCondition
	} else {
		sql = fmt.Sprintf(
			`select count(1) 
			   from ( select  p.project_id %s  union select  p.project_id  
			   from project p 
			   left join project_member pm on p.project_id = pm.project_id
			   left join user_group ug on ug.id = pm.entity_id and pm.entity_type = 'g' and ug.group_type = 1
			   where ug.ldap_group_dn in ( %s )) t`,
			sqlCondition, groupDNCondition)
	}
	log.Debugf("query sql:%v", sql)
	var count int
	if err := GetOrmer().Raw(sql, params).QueryRow(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func projectQueryConditions(query *models.ProjectQueryParam) (string, []interface{}) {
	params := []interface{}{}
	sql := ` from project as p`
	if query == nil {
		sql += ` where p.deleted=false`
		return sql, params
	}
	// if query.ProjectIDs is not nil but has no element, the query will returns no rows
	if query.ProjectIDs != nil && len(query.ProjectIDs) == 0 {
		sql += ` where 1 = 0`
		return sql, params
	}
	if len(query.Owner) != 0 {
		sql += ` join harbor_user u1
					on p.owner_id = u1.user_id`
	}
	if query.Member != nil && len(query.Member.Name) != 0 {
		sql += ` join project_member pm
					on p.project_id = pm.project_id and pm.entity_type = 'u'
					join harbor_user u2
					on pm.entity_id=u2.user_id`
	}
	sql += ` where p.deleted=false`

	if len(query.Owner) != 0 {
		sql += ` and u1.username=?`
		params = append(params, query.Owner)
	}

	if len(query.Name) != 0 {
		sql += ` and p.name like ?`
		params = append(params, "%"+Escape(query.Name)+"%")
	}

	if query.Member != nil && len(query.Member.Name) != 0 {
		sql += ` and u2.username=?`
		params = append(params, query.Member.Name)

		if query.Member.Role > 0 {
			sql += ` and pm.role = ?`
			roleID := 0
			switch query.Member.Role {
			case common.RoleProjectAdmin:
				roleID = 1
			case common.RoleDeveloper:
				roleID = 2
			case common.RoleGuest:
				roleID = 3
			case common.RoleMaster:
				roleID = 4
			}
			params = append(params, roleID)
		}
	}
	if len(query.ProjectIDs) > 0 {
		sql += fmt.Sprintf(` and p.project_id in ( %s )`,
			paramPlaceholder(len(query.ProjectIDs)))
		params = append(params, query.ProjectIDs)
	}
	return sql, params
}

// CreatePagination ...
func CreatePagination(query *models.ProjectQueryParam, sql string, params []interface{}) (string, []interface{}) {
	if query != nil && query.Pagination != nil && query.Pagination.Size > 0 {
		sql += ` limit ?`
		params = append(params, query.Pagination.Size)

		if query.Pagination.Page > 0 {
			sql += ` offset ?`
			params = append(params, (query.Pagination.Page-1)*query.Pagination.Size)
		}
	}
	return sql, params
}

// DeleteProject ...
func DeleteProject(id int64) error {
	project, err := GetProjectByID(id)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s#%d", project.Name, project.ProjectID)
	sql := `update project 
		set deleted = true, name = ? 
		where project_id = ?`
	_, err = GetOrmer().Raw(sql, name, id).Exec()
	return err
}

// GetRolesByLDAPGroup - Get Project roles of the
// specified group DN is a member of current project
func GetRolesByLDAPGroup(projectID int64, groupDNCondition string) ([]int, error) {
	var roles []int
	if len(groupDNCondition) == 0 {
		return roles, nil
	}
	o := GetOrmer()
	// Because an LDAP user can be memberof multiple groups,
	// the role is in descent order (1-admin, 2-developer, 3-guest, 4-master), use min to select the max privilege role.
	sql := fmt.Sprintf(
		`select min(pm.role) from project_member pm 
		left join user_group ug on pm.entity_type = 'g' and pm.entity_id = ug.id 
		where ug.ldap_group_dn in ( %s ) and pm.project_id = ? `,
		groupDNCondition)
	log.Debugf("sql:%v", sql)
	if _, err := o.Raw(sql, projectID).QueryRows(&roles); err != nil {
		log.Warningf("Error in GetRolesByLDAPGroup, error: %v", err)
		return nil, err
	}
	// If there is no row selected, the min returns an empty row, to avoid return 0 as role
	if len(roles) == 1 && roles[0] == 0 {
		return []int{}, nil
	}
	return roles, nil
}
