import { Injectable, CanActivate, ExecutionContext, UnauthorizedException, ForbiddenException } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { ROLES_KEY } from './roles.decorator';

@Injectable()
export class RolesGuard implements CanActivate {
  constructor(private reflector: Reflector) {}
  canActivate(context: ExecutionContext): boolean {
    const roles = this.reflector.getAllAndOverride<string[]>(ROLES_KEY, [context.getHandler(), context.getClass()]);
    if (!roles || roles.length === 0) return true;
    const req = context.switchToHttp().getRequest();
    const user = (req as any).user;
    if (!user) throw new UnauthorizedException();
    if (!roles.includes(user.role)) throw new ForbiddenException();
    return true;
  }
}

