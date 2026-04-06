// Validate email format
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

// Validate username (3-20 chars, alphanumeric and underscore)
export function isValidUsername(username: string): boolean {
  const usernameRegex = /^[a-zA-Z0-9_]{3,20}$/;
  return usernameRegex.test(username);
}

// Validate password (at least 6 chars)
export function isValidPassword(password: string): boolean {
  return password.length >= 6;
}

// Validate registration form
export function validateRegisterForm(
  username: string,
  email: string,
  password: string
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!isValidUsername(username)) {
    errors.push('用户名需要 3-20 个字符，只能包含字母、数字和下划线');
  }

  if (!isValidEmail(email)) {
    errors.push('邮箱格式不正确');
  }

  if (!isValidPassword(password)) {
    errors.push('密码至少需要 6 个字符');
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// Validate login form
export function validateLoginForm(
  account: string,
  password: string
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!account.trim()) {
    errors.push('请输入用户名或邮箱');
  }

  if (!password) {
    errors.push('请输入密码');
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// Validate update user form
export function validateUpdateUserForm(
  username: string,
  email: string
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!isValidUsername(username)) {
    errors.push('用户名需要 3-20 个字符，只能包含字母、数字和下划线');
  }

  if (!isValidEmail(email)) {
    errors.push('邮箱格式不正确');
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}