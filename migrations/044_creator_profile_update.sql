-- Migration 044: Atualizar perfil do Criador (Jose R F Junior)
-- CPF: 64525430249, ID: 1121
-- Persona: psychologist (doce), Voice: Aoede, Nivel: super_genio

UPDATE idosos SET
    persona_preferida = 'psychologist',
    nivel_cognitivo = 'super_genio',
    tom_voz = 'doce_maximo',
    voice_name = 'Aoede',
    estilo_conversa = 'profundo',
    profundidade_emocional = 1.0
WHERE cpf = '64525430249';
