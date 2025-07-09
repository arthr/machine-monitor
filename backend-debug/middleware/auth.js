/**
 * Middleware de autenticação
 */

const config = require("../config/config");
const logger = require("../utils/logger");

/**
 * Middleware para verificar token de autenticação
 */
function authenticate(req, res, next) {
	const token = req.headers.authorization;

	if (token !== `Bearer ${config.auth.token}`) {
		logger.warning(`Tentativa de acesso não autorizada: ${token}`);
		return res.status(401).json({
			error: "Token inválido",
			message: "Acesso negado",
		});
	}

	logger.debug("Token de autenticação válido");
	next();
}

module.exports = {
	authenticate,
};
