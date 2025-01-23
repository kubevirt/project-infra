# CANNIER

Implementation of a feature set extraction as described in the [CANNIER] paper.

## Table 1: The 18 features measured by pytest-CANNIER.<sup>[CANNIER] p. 9</sup>

| Number | Name                  | Description                                                                       |
|--------|-----------------------|-----------------------------------------------------------------------------------|
| 1      | Read Count            | Number of times the filesystem had to perform input                               |
| 2      | Write Count           | Number of times the filesystem had to perform output                              |
| 3      | Run Time              | Elapsed wall-clock time of the whole test case execution.                         |
| 4      | Wait Time             | Elapsed wall-clock time spent waiting for input/output operations to complete.    |
| 5      | Context Switches      | Number of voluntary context switches.                                             |
| 6      | Covered Lines         | Number of lines covered.                                                          |
| 7      | Source Covered Lines  | Number of lines covered that are not part of test cases.                          |
| 8      | Covered Changes       | Total number of times each covered line has been modified in the last 75 commits. |
| 9      | Max. Threads          | Peak number of concurrently running threads.                                      |
| 10     | Max. Children         | Peak number of concurrently running child processes.                              |
| 11     | Max. Memory           | Peak memory usage.                                                                |
| 12     | AST Depth             | Maximum depth of nested program statements in the test case code.                 |
| 13     | Assertions            | Number of assertion statements in the test case code.                             |
| 14     | External Modules      | Number of non-standard modules (i.e., libraries) used by the test case.           |
| 15     | Halstead Volume       | A measure of the size of an algorithm’s implementation [21, 57, 59].              |
| 16     | Cyclomatic Complexity | Number of branches in the test case code [39, 57, 59].                            |
| 17     | Test Lines of Code    | Number of lines in the test case code [57, 59].                                   |
| 18     | Maintainability       | A measure of how easy the test case code is to support and modify [19, 71].       |

[CANNIER]: https://www.gregorykapfhammer.com/download/research/papers/key/Parry2023-paper.pdf