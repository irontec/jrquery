jquery is simple CLI tool to display Jira tickets on terminal

## Contributing

All lines of code for this tool have been created by ChatGPT, so most probably there will be lots of possible enhancements.
Don't hesitate to open a pull request with any suggestion.

## Configuration

During the first run, jquery will request some configuration data:

 - Jira BaseURL (p.e. https://irontec.atlassian.net)
 - Jira Login Email (p.e. kaian@irontec.com)
 - Jira API Token (you can generate a token at https://id.atlassian.com/manage-profile/security/api-tokens)


## Usage

When run without parameters, jquery displays current user unresolved issues.

Please refer to the help section for additional query parameters.

```
Usage:
  jquery [OPTIONS]

Application Options:
  -d, --debug          Print debugging information
  -u, --user=          Name or email of assigned user
  -p, --project=       Key of project to search issues
  -s, --search         Search text in summary, issue description or comments
  -l, --limit=         Limit output to first N results (default: 50)
  -c, --count          Only print issue count
  -S, --sprint         Only print issues with active sprint
  -e, --status=        Only print issues with given status Name
  -O, --unresolved     Only print unresolved issues
  -A, --all            Print all issues no matter their status
  -q, --query=         Run a custom query
  -f, --filter=        Search issues using a saved Jira filter ID
  -o, --open=          Open given issue in a browser tab
  -T, --order-by-time  Sort issues by last updated time (use -TT for reverse)
  -U, --order-by-user  Sort issues by assignee (use -UU for reverse ordering)
      --list-projects  List all visible projects for current user
      --list-users     List all users in Jira
      --list-filters   List all saved filters in Jira

Help Options:
  -h, --help           Show this help message
```

## License
    jquery - Jira Issues query tool
    Copyright (C) 2024 Irontec S.L.

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    In addition, as a special exception, the copyright holders give
    permission to link the code of portions of this program with the
    OpenSSL library under certain conditions as described in each
    individual source file, and distribute linked combinations
    including the two.
    You must obey the GNU General Public License in all respects
    for all of the code used other than OpenSSL.  If you modify
    file(s) with this exception, you may extend this exception to your
    version of the file(s), but you are not obligated to do so.  If you
    do not wish to do so, delete this exception statement from your
    version.  If you delete this exception statement from all source
    files in the program, then also delete it here.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.