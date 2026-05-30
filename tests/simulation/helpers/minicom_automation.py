#!/usr/bin/env python3
"""
ShareSerial 终端自动化脚本
使用 pexpect 自动化 minicom/picocom 测试

使用方法:
    python3 minicom_automation.py start /tmp/vttyTest0
    python3 minicom_automation.py test /tmp/vttyTest0 "command" "expected_output"
    python3 minicom_automation.py read /tmp/vttyTest0 timeout_seconds
"""

import pexpect
import sys
import time
import json
import os

class TerminalAutomation:
    def __init__(self, pty_path, terminal='minicom'):
        self.pty_path = pty_path
        self.terminal = terminal
        self.child = None
        self.buffer = []
        self.started = False

    def start(self):
        """启动终端程序"""
        try:
            if self.terminal == 'minicom':
                # minicom 启动
                self.child = pexpect.spawn('minicom -D {}'.format(self.pty_path),
                                          encoding='utf-8',
                                          timeout=10)
                # 等待 minicom 欢迎信息
                self.child.expect('Press CTRL-A', timeout=10)
                self.started = True
                return {'status': 'started', 'message': 'minicom started successfully'}

            elif self.terminal == 'picocom':
                # picocom 启动
                self.child = pexpect.spawn('picocom {}'.format(self.pty_path),
                                          encoding='utf-8',
                                          timeout=10)
                self.child.expect('Terminal ready', timeout=10)
                self.started = True
                return {'status': 'started', 'message': 'picocom started successfully'}

            elif self.terminal == 'cat':
                # 简单的 cat 模式（只读取）
                self.child = pexpect.spawn('cat {}'.format(self.pty_path),
                                          encoding='utf-8',
                                          timeout=30)
                self.started = True
                return {'status': 'started', 'message': 'cat started successfully'}

            else:
                return {'status': 'error', 'message': 'Unknown terminal: {}'.format(self.terminal)}

        except pexpect.TIMEOUT:
            return {'status': 'error', 'message': 'Timeout starting terminal'}
        except Exception as e:
            return {'status': 'error', 'message': str(e)}

    def send_command(self, command):
        """发送命令"""
        if not self.started or self.child is None:
            return {'status': 'error', 'message': 'Terminal not started'}

        try:
            self.child.sendline(command)
            self.buffer.append({'type': 'sent', 'data': command, 'time': time.time()})
            return {'status': 'sent', 'command': command}
        except Exception as e:
            return {'status': 'error', 'message': str(e)}

    def wait_for_output(self, pattern, timeout=5):
        """等待特定输出"""
        if not self.started or self.child is None:
            return {'status': 'error', 'message': 'Terminal not started'}

        try:
            self.child.expect(pattern, timeout=timeout)
            output = self.child.before
            self.buffer.append({'type': 'received', 'data': output, 'time': time.time()})
            return {'status': 'found', 'output': output}
        except pexpect.TIMEOUT:
            return {'status': 'timeout', 'message': 'Pattern not found within timeout'}
        except Exception as e:
            return {'status': 'error', 'message': str(e)}

    def read_output(self, timeout=2):
        """读取当前输出"""
        if not self.started or self.child is None:
            return {'status': 'error', 'message': 'Terminal not started'}

        try:
            # 使用超时读取
            self.child.expect(pexpect.EOF, timeout=timeout)
            output = self.child.before
            return {'status': 'read', 'output': output}
        except pexpect.TIMEOUT:
            # 超时时返回已读取的内容
            return {'status': 'read', 'output': self.child.before or ''}
        except Exception as e:
            return {'status': 'error', 'message': str(e)}

    def get_buffer(self):
        """获取所有缓冲数据"""
        return {'status': 'ok', 'buffer': self.buffer}

    def close(self):
        """关闭终端"""
        if not self.started or self.child is None:
            return {'status': 'ok', 'message': 'Already closed'}

        try:
            if self.terminal == 'minicom':
                # minicom 退出: Ctrl-A X
                self.child.sendcontrol('a')
                self.child.send('x')
                time.sleep(0.5)

            self.child.close()
            self.started = False
            return {'status': 'closed', 'message': 'Terminal closed'}
        except Exception as e:
            self.started = False
            return {'status': 'error', 'message': str(e)}


def main():
    """CLI 入口"""
    if len(sys.argv) < 3:
        print(json.dumps({'status': 'error', 'message': 'Usage: python3 minicom_automation.py <action> <pty_path> [args...]'}))
        sys.exit(1)

    action = sys.argv[1]
    pty_path = sys.argv[2]
    terminal = sys.argv[3] if len(sys.argv) > 3 else 'minicom'

    auto = TerminalAutomation(pty_path, terminal)

    result = None

    if action == 'start':
        result = auto.start()
        if result['status'] == 'started':
            print(json.dumps(result))
            # 保持运行，等待后续命令
            try:
                while auto.started:
                    auto.read_output(1)
                    time.sleep(0.5)
            except KeyboardInterrupt:
                auto.close()
        else:
            print(json.dumps(result))

    elif action == 'test':
        if len(sys.argv) < 5:
            print(json.dumps({'status': 'error', 'message': 'Usage: test <pty_path> <command> <expected_pattern>'}))
            sys.exit(1)

        command = sys.argv[3]
        expected = sys.argv[4]

        # 启动
        start_result = auto.start()
        if start_result['status'] != 'started':
            print(json.dumps(start_result))
            sys.exit(1)

        time.sleep(1)

        # 发送命令
        auto.send_command(command)

        # 等待输出
        result = auto.wait_for_output(expected, timeout=5)

        # 关闭
        auto.close()

        print(json.dumps(result))

    elif action == 'read':
        timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 5

        start_result = auto.start()
        if start_result['status'] != 'started':
            print(json.dumps(start_result))
            sys.exit(1)

        time.sleep(1)

        result = auto.read_output(timeout)
        auto.close()

        print(json.dumps(result))

    elif action == 'verify':
        expected = sys.argv[3] if len(sys.argv) > 3 else ''

        start_result = auto.start()
        if start_result['status'] != 'started':
            print(json.dumps(start_result))
            sys.exit(1)

        time.sleep(2)

        # 读取输出
        result = auto.read_output(3)

        # 验证是否包含期望内容
        if expected and expected in result.get('output', ''):
            result['contains_expected'] = True
        else:
            result['contains_expected'] = False

        auto.close()

        print(json.dumps(result))

    else:
        print(json.dumps({'status': 'error', 'message': 'Unknown action: {}'.format(action)}))
        sys.exit(1)


if __name__ == '__main__':
    main()